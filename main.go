package main

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type Cars map[string]map[string]map[string][]map[string]string

// проверка есть элемент item в слайсе slice
func CheckItem(slice []string, item string) (bool, int) {
	for i, e := range slice {
		if item == e {
			return true, i
		}
	}
	return false, -1
}

// добавление машины
func AddCar(markName, modelName, year, pictureURL, carURL string, marks Cars) {
	if models, ok := marks[markName]; ok {
		if years, ok2 := models[modelName]; ok2 {
			if yearMaps, ok3 := years[year]; ok3 {
				models[modelName][year] = append(yearMaps, map[string]string{pictureURL: carURL})
			} else {
				models[modelName][year] = []map[string]string{{pictureURL: carURL}}
			}
		} else {
			models[modelName] = make(map[string][]map[string]string)
			models[modelName][year] = []map[string]string{{pictureURL: carURL}}
		}
	} else {
		marks[markName] = make(map[string]map[string][]map[string]string)
		marks[markName][modelName] = make(map[string][]map[string]string)
		marks[markName][modelName][year] = []map[string]string{{pictureURL: carURL}}
	}
}

// структура для конфига
type Config struct {
	ParseHTTP bool `json:"parseHTTP"`
	URL string `json:"url"`
	UseProxy bool `json:"useProxy"`
	Proxy string `json:"Proxy"`
}

// получение конфига
func GetConfig(filename string) (config Config){
	f, err := os.Open(filename)
	if err != nil {
		log.Print(err)
		return Config{true, "", false, ""}
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)

	json.Unmarshal(b, &config)
	return
}

// парсинг ссылки
func GetDataHTTP(URL string, data Cars, proxyURL *url.URL) {
	basicURL := "https://auto.ria.com/uk/search/?"

	u, _ := url.Parse(URL)
	m, _ := url.ParseQuery(u.RawQuery)
	m.Set("size", "100")
	page := 0
	pageStr := strconv.Itoa(page)
	m.Set("page", pageStr)
	URL = basicURL + m.Encode()

	var (
		client *http.Client
		resp *http.Response
		err error
	)
	if proxyURL != nil {
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	var i int = 1
	for i != 0{
		if client != nil {
			resp, err = client.Get(URL)
		} else {
			resp, err = http.Get(URL)
		}
		if err != nil {
			log.Panic(err)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Panic(err)
		}

		i = 0
		doc.Find("#searchResults section").Each(func(index int, item *goquery.Selection) {
			markName := item.Find(".hide").AttrOr("data-mark-name", "")
			modelName := item.Find(".hide").AttrOr("data-model-name", "")
			year := item.Find(".hide").AttrOr("data-year", "")
			pictureURL := item.Find(".content-bar .ticket-photo picture img").AttrOr("src", "")
			carURL := item.Find(".content-bar .m-link-ticket").AttrOr("href", "")
			if markName != "" && modelName != "" && year != "" && pictureURL != "" && carURL != "" {
				AddCar(markName, modelName, year, pictureURL, carURL, data)
				i++
			}
		})

		if i != 0 {
			page++
			pageStr = strconv.Itoa(page)
			m.Set("page", pageStr)
			URL = basicURL + m.Encode()
			//fmt.Println(page)
		}
		resp.Body.Close()
	}
}

// получение готовых данных из файла
func GetDataFile(filename string, data Cars) {
	f, err := os.Open(filename)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)

	json.Unmarshal(b, &data)
	return
}

// получение фильтра
func GetExclude(filename string) (exclude map[string]map[string][]string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()
	b, _ := ioutil.ReadAll(f)

	json.Unmarshal(b, &exclude)
	return
}

// применение фильтра
func ApplyFilter(data Cars, exclude map[string]map[string][]string) {
	for keyMark, valueModels := range exclude {
		if _, okMark := data[keyMark]; len(valueModels) == 0 && okMark {
			delete(data, keyMark)
		} else {
			for keyModel, valueYears := range valueModels {
				if _, okModel := data[keyMark][keyModel]; len(valueYears) == 0 && okModel {
					delete(data[keyMark], keyModel)
				} else {
					for _, keyYear := range valueYears {
						if _, okYear := data[keyMark][keyModel][keyYear]; okYear {
							delete(data[keyMark][keyModel], keyYear)
						}
					}
					if len(data[keyMark][keyModel]) == 0 {
						delete(data[keyMark], keyModel)
					}
				}
			}
			if len(data[keyMark]) == 0 {
				delete(data, keyMark)
			}
		}
	}
}

// запись полученных данных в файл
func WriteToFile(filename string, data Cars) {
	f, err := os.Create("./" + filename)

	if err != nil {
		panic(err)
	}
	defer f.Close()

	jsonData, _ := json.Marshal(data)
	f.Write(jsonData)
}

func main() {
	config := GetConfig("config.json")
	data := make(Cars)

	if config.ParseHTTP {
		if config.UseProxy {
			proxyURL, err := url.Parse(config.Proxy)
			if err != nil {
				log.Panic(err)
			}
			GetDataHTTP(config.URL, data, proxyURL)

		} else {
			GetDataHTTP(config.URL, data, nil)
		}
	} else {
		GetDataFile("input.json", data)
	}


	exclude := GetExclude("exclude.json")
	ApplyFilter(data, exclude)

	WriteToFile("output.json", data)
}
