package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RequestData struct {
	ProgramType  string `json:"program_type"`
	ProgramLangs string `json:"program_langs"`
	Data         struct {
		PrefName string `json:"prefname"`
		CityName string `json:"cityname"`
	} `json:"data"`
}

type ResponseData struct {
	ProgramType  string `json:"program_type"`
	ReturnType   string `json:"return_type"`
	ResponseCode string `json:"Responce_Code"`
	Body         any    `json:"body"`
	Response     string `json:"response"`
}

func main() {
	http.HandleFunc("/weather", weatherHandler)
	log.Println("Starting server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONResponse(w, http.StatusMethodNotAllowed, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error",
			ResponseCode: "405 Method Not Allowed",
			Body: map[string]string{
				"detail": "Only POST method is allowed.",
			},
			Response: "failure",
		})
		return
	}

	var requestData RequestData
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		writeJSONResponse(w, http.StatusBadRequest, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error",
			ResponseCode: "400 Bad Request",
			Body: map[string]string{
				"detail": "Invalid JSON format.",
			},
			Response: "failure",
		})
		return
	}

	prefName := strings.TrimSpace(requestData.Data.PrefName)
	cityName := strings.TrimSpace(requestData.Data.CityName)

	cityIDs, err := loadCityIDs("./city_ids.json")
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error",
			ResponseCode: "500 Internal Server Error",
			Body: map[string]string{
				"detail": fmt.Sprintf("Failed to load city_ids.json: %v", err),
			},
			Response: "failure",
		})
		return
	}

	if prefName == "" && cityName == "" {
		writeJSONResponse(w, http.StatusOK, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "data_is_None",
			ResponseCode: "200 OK",
			Body: map[string]string{
				"city_ids.json": string(flattenJSON(cityIDs)),
			},
			Response: "success",
		})
		return
	}

	if cityName != "" && prefName == "" {
		writeJSONResponse(w, http.StatusBadRequest, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error",
			ResponseCode: "400 Bad Request",
			Body: map[string]string{
				"detail": "If the object \"perfname\" is not stored in \"data\", it is not allowed to store \"cityname\".",
			},
			Response: "failure",
		})
		return
	}

	if prefName != "" && cityName == "" {
		cityMap, exists := cityIDs[prefName]
		if !exists {
			writeJSONResponse(w, http.StatusNotFound, ResponseData{
				ProgramType:  "Get_Weather",
				ReturnType:   "Error",
				ResponseCode: "404 Not Found",
				Body: map[string]string{
					"detail": "Prefecture not found.",
				},
				Response: "failure",
			})
			return
		}
		writeJSONResponse(w, http.StatusOK, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "cityname_is_None",
			ResponseCode: "200 OK",
			Body: map[string]string{
				"detail":   "Correct request format, but the value \"cityname\" is None.",
				"cityname": fmt.Sprintf("%v", cityMap),
			},
			Response: "success",
		})
		return
	}

	cityMap, exists := cityIDs[prefName]
	if !exists {
		writeJSONResponse(w, http.StatusNotFound, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error",
			ResponseCode: "404 Not Found",
			Body: map[string]string{
				"detail": "Prefecture not found.",
			},
			Response: "failure",
		})
		return
	}

	cityID, exists := cityMap[cityName]
	if !exists {
		writeJSONResponse(w, http.StatusNotFound, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error",
			ResponseCode: "404 Not Found",
			Body: map[string]string{
				"detail": "City not found in the specified prefecture.",
			},
			Response: "failure",
		})
		return
	}

	url := fmt.Sprintf("https://weather.tsukumijima.net/api/forecast/city/%s", cityID)
	resp, err := http.Get(url)
	if err != nil {
		writeJSONResponse(w, http.StatusGatewayTimeout, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "API Server Error",
			ResponseCode: "504 Gateway Timeout",
			Body: map[string]string{
				"detail": "We waited for the specified time set by this server's code, but received no response.",
			},
			Response: "failure",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		writeJSONResponse(w, http.StatusBadGateway, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error of any kind",
			ResponseCode: "502 Bad Gateway",
			Body: map[string]string{
				"detail":                  "I sent a request to the API server and received a response but an error was returned.",
				"api_server_error_detail": string(body),
			},
			Response: "failure",
		})
		return
	}

	apiBody, _ := ioutil.ReadAll(resp.Body)
	var formattedJSON map[string]interface{}
	if err := json.Unmarshal(apiBody, &formattedJSON); err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, ResponseData{
			ProgramType:  "Get_Weather",
			ReturnType:   "Error",
			ResponseCode: "500 Internal Server Error",
			Body: map[string]string{
				"detail": "Failed to format JSON from API response.",
			},
			Response: "failure",
		})
		return
	}

	formattedBytes, _ := json.MarshalIndent(formattedJSON, "", "  ")
	writeJSONResponse(w, http.StatusOK, ResponseData{
		ProgramType:  "Get_Weather",
		ReturnType:   "Completed successfully",
		ResponseCode: "200 OK",
		Body: map[string]string{
			"detail":    "Normal operation was returned from the API.",
			"main_data": string(formattedBytes),
		},
		Response: "success",
	})
}

func loadCityIDs(filePath string) (map[string]map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var cityIDs map[string]map[string]string
	if err := json.Unmarshal(bytes, &cityIDs); err != nil {
		return nil, err
	}

	return cityIDs, nil
}

func flattenJSON(data map[string]map[string]string) []byte {
	bytes, _ := json.Marshal(data)
	return bytes
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, response ResponseData) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func logUnknownError(err error) {
	timestamp := time.Now().Format("06-01-02_15-04-05")
	errorFilePath := filepath.Join("./Error_Logs", fmt.Sprintf("Unknown_Error-%s.txt", timestamp))
	_ = os.MkdirAll("./Error_Logs", 0755)
	file, fileErr := os.Create(errorFilePath)
	if fileErr != nil {
		log.Printf("Failed to create error log file: %v", fileErr)
		return
	}
	defer file.Close()
	file.WriteString(err.Error())
}
