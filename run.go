package main

import (
  "net/http"
  "encoding/json"
  "bytes"
  "os"
  "fmt"
  "io/ioutil"
  "crypto/sha256"
  "encoding/hex"
  "log"
  "time"
  "os/exec"
)

var API_URL string = os.Getenv("API_URL")
var PHONE string = os.Getenv("PHONE")

var transactionID string
var done = make(chan bool)
var token string

func generateOTP() {

  body, _ := json.Marshal(map[string]string{
      "mobile":  PHONE,
   })
  encodedBody := bytes.NewBuffer(body)
  resp, err := http.Post(API_URL + "/v2/auth/public/generateOTP", "application/json", encodedBody)
  if err != nil {
    fmt.Println("An error occured while calling generateOTP")
    panic(err)
  }
  defer resp.Body.Close()
  readBody, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    fmt.Println("An error occured while reading the response of generateOTP")
    panic(err)
  }
  var decodedResp map[string]string
  json.Unmarshal(readBody, &decodedResp)
  transactionID = decodedResp["txnId"]
  fmt.Println("Transaction ID:", transactionID)
}

func confirmOTP(otp string) string {
  body, _ := json.Marshal(map[string]string{
      "otp":  encodeOTP(otp),
      "txnId": transactionID,
   })
  encodedBody := bytes.NewBuffer(body)
  resp, err := http.Post(API_URL + "/v2/auth/public/confirmOTP", "application/json", encodedBody)
  if err != nil {
    fmt.Println("An error occured while calling confirmOTP")
    panic(err)
  }
  defer resp.Body.Close()
  readBody, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    fmt.Println("An error occured while reading the response of confirmOTP")
    panic(err)
  }
  log.Println(string([]byte(readBody)))
  var decodedResp map[string]string
  json.Unmarshal(readBody, &decodedResp)
  token = decodedResp["token"]
  fmt.Println("Authentication successful. Token:", token)
  return token
}

func encodeOTP(otp string) string {
  sum := sha256.Sum256([]byte(otp))
  return hex.EncodeToString(sum[:])
}

func calendarByDistrict(districtID string, date string) {
  params := "?district_id=" + districtID + "&date=" + date
  url := API_URL + "/v2/appointment/sessions/public/calendarByDistrict" + params
  fmt.Println("Calling URL:", url)
  req, err := http.NewRequest("GET", url, nil)
  bearer := "bearer " + token
  req.Header.Add("Authorization", bearer)
  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    log.Println("Error on response.\n[ERROR] -", err)
  }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    log.Println("Error while reading the response bytes:", err)
  }
  //log.Println(string([]byte(body)))
  if (resp.StatusCode >= 300 || resp.StatusCode < 200) {
    log.Println(string([]byte(body)))
    errorMessage := fmt.Sprintf("Authorization required, StatusCode: %v", resp.StatusCode)
    panic(errorMessage)
  }
  var response CalendarResponse
  json.Unmarshal(body, &response)
  centers := response.Centers
  fmt.Println("Total centers:", len(centers))
  for _, center := range centers {
    if (center.Sessions != nil) {
      for _, session := range center.Sessions {
        if (session.MinAgeLimit >= 18 && session.MinAgeLimit < 45 && session.AvailableCapacity > 0) {
          exec.Command("say", "slot found").Output()
          fmt.Printf("%v slot available at %v", session.AvailableCapacity, center.Name)
          fmt.Printf("More details about the center: %+v\n", center)
        }
      }
    }
  }
}

func getCall(url string, bearer string) {
  req, _ := http.NewRequest("GET", url, nil)
  req.Header.Add("Authorization", "Bearer " + token)
  client := &http.Client{}
  resp, err := client.Do(req)

  if err != nil {
    log.Fatalln("Error while making request " + url)
  }

  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  log.Println(string([]byte(body)))
}

func createURL(districtID string, date string) string {
  params := "?district_id=" + districtID + "&date=" + date
  return API_URL + "/v2/appointment/sessions/public/calendarByDistrict" + params
}


type Session struct {
  SessionID int32 `json:"session_id"`
  Date string `json:"date"`
  AvailableCapacity int32 `json:"available_capacity"`
  MinAgeLimit int32 `json:"min_age_limit"`
  Vaccine string `json:"vaccine"`
  Slots []string `json:"slots"`
}



type Centers struct {
  CenterID int32 `json:"center_id"`
  Name string `json:"name"`
  Address string `json:"address"`
  StateName string `json:"state_name"`
  DistrictName string `json:"district_name"`
  BlockName string `json:"block_name"`
  Pincode string `json:"pincode"`
  Lat string `json:"lat"`
  Long string `json:"long"`
  From string `json:"from"`
  To string `json:"to"`
  FeeType string `json:"fee_type"`
  Sessions []Session `json:"sessions"`
}

type CalendarResponse struct {
  Centers []Centers `json:"centers"`
}

func runEvery(seconds int, params map[string]string) {
  ticker := time.NewTicker(time.Duration(seconds) * time.Second)
  defer ticker.Stop()
  go func() {
		time.Sleep(900 * time.Second)
		done <- true
	}()

  for {
		select {
		case <-done:
			fmt.Println("Done!")
			return
		case t := <-ticker.C:
      fmt.Println(t)
      calendarByDistrict(params["districtID"], params["date"])
		}
	}
}

func main() {
  generateOTP()
  fmt.Println("Please enter the OTP received on your phone:")
  var otp string
  fmt.Scanln(&otp)
  x := confirmOTP(otp)
  var districtID string
  var date string
  fmt.Println("Please enter the district ID:")
  fmt.Scanln(&districtID)
  fmt.Println("Please enter the starting date of the week you want to check for:")
  fmt.Scanln(&date)
  getCall(createURL(districtID, date), x)
  //m := make(map[string]string)
  //m["districtID"] = districtID
  //m["date"] = date
  //runEvery(1, m)
}
