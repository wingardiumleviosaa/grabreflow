package service

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

type Service struct {
	convergence     string
	reflowserver    string
	reflowParameter *reflowParameter
	//driver          string
}

type reflowParameter struct {
	line       string
	start_date string
	end_date   string
	sn         string
}

func NewService() *Service {
	s := new(Service)
	return s
}

func (s *Service) Init(ctx context.Context) error {
	s.reflowParameter = new(reflowParameter)

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("config")
	viper.AddConfigPath("../../config")

	err := viper.ReadInConfig()
	if err != nil {
		log.Panic("Fatal error config file:", err)
	}

	s.convergence = viper.GetString("API_service.convergence")
	s.reflowserver = viper.GetString("API_service.reflowserver")

	//s.driver = viper.GetString("Chrome.driver_path")

	return nil
}

func (s *Service) GrabReflow(c *gin.Context) {

	// Get reflow info from convergence api
	s.reflowParameter.sn = c.Param("sn")
	dt := time.Now()
	s.reflowParameter.end_date = fmt.Sprintf(dt.Format("01/02/2006 03:04 PM")) // set end date to current time

	convurl := s.convergence + s.reflowParameter.sn
	resp, err := http.Get(convurl)
	getReturn := make(map[string]interface{})

	if err == nil {
		if resp.StatusCode == 200 {
			//////// 1. get the target product's info (intime & line)////////
			// result[sn] = traceability_website
			body, _ := ioutil.ReadAll(resp.Body)
			_ = json.Unmarshal(body, &getReturn) // parse return byte data
			machineArr := getReturn["history"].([]interface{})
			for _, machine := range machineArr { // range the machine list to get line and in_time
				m := machine.(map[string]interface{})
				switch m["id"] {
				case "SMT_A_M6":
					intime := int64(m["in_time"].(float64))
					t := time.Unix(intime, 0)
					s.reflowParameter.start_date = fmt.Sprintf("%s 12:00 AM", t.Format("01/02/2006"))
					s.reflowParameter.line = "6" // A line

				case "SMT_B_M7":
					intime := int64(m["in_time"].(float64))
					t := time.Unix(intime, 0)
					s.reflowParameter.start_date = fmt.Sprintf("%s 12:00 AM", t.Format("01/02/2006"))
					s.reflowParameter.line = "855187" // B line
				}
			}
			//////// 2. Enable selenium service ////////
			opts := []selenium.ServiceOption{
				selenium.Output(os.Stderr),
			}
			// new a webdriver's instance
			service, err := selenium.NewChromeDriverService("/usr/bin/chromedriver", 9515, opts...)
			// service, err := selenium.NewChromeDriverService("./chromedriver", 9515, opts...)
			if err != nil {
				log.Error("Error starting the ChromeDriver server:", err)
			}
			// delay service shutdown
			// defer service.Stop()

			//////// 3. Call browser ////////
			caps := selenium.Capabilities{
				"browserName": "chrome",
			}
			// set chrome arguments
			chromeCaps := chrome.Capabilities{
				Args: []string{
					"--headless",   // do not open the browser
					"--no-sandbox", // allow non-root to execute Chrome
					"--disable-deb-shm-usage",
					"--disable-gpu", // in order to avoid https://issueexplorer.com/issue/microsoft/playwright/9788
					"--window-size=1920,1440",
					// "--start-maximized", // maximize the windows, 雖然如此但用這個跑比例會不是全螢幕
					"--allowd-ips", // in order to avoid "bind() failed: cannot assign request address (99)" error, refer to https://reurl.cc/QjbO05 & https://www.jianshu.com/p/65cd4b138ee8
					// "--verbose", // in order to avoid "bind() failed: cannot assign request address (99)" error
				},
			}
			caps.AddChrome(chromeCaps)

			// connect to the local webdriver's instance
			wd, err := selenium.NewRemote(caps, "http://127.0.0.1:9515/wd/hub")
			if err != nil {
				log.Fatal("connect to the webDriver faild:", err)
			}
			// defer wd.Quit()

			//////// 4. Connect to the target website and manipulate the web element ////////
			if err := wd.Get(s.reflowserver); err != nil {
				log.Fatal("connect to the reflow server failed:", err)
			}
			// select the line
			sel_line, _ := wd.FindElement(selenium.ByXPATH, "/html/body/div/div/div/div/section[2]/div[1]/div/div/div[2]/div[1]/select")
			line, _ := selenium.Select(sel_line)
			line.SelectByValue(s.reflowParameter.line)

			// select start date
			start_date, _ := wd.FindElement(selenium.ByXPATH, "/html/body/div/div/div/div/section[2]/div[1]/div/div/div[2]/div[2]/table/tbody/tr[2]/td[1]/div/input")
			start_date.Clear()
			start_date.SendKeys(s.reflowParameter.start_date)

			// select end date
			end_date, _ := wd.FindElement(selenium.ByXPATH, "/html/body/div/div/div/div/section[2]/div[1]/div/div/div[2]/div[2]/table/tbody/tr[2]/td[2]/div/input")
			end_date.Clear()
			end_date.SendKeys(s.reflowParameter.end_date)

			// input serial number
			barcode, _ := wd.FindElement(selenium.ByXPATH, "/html/body/div/div/div/div/section[2]/div[1]/div/div/div[2]/table/tbody/tr[2]/td[3]/input")
			barcode.SendKeys(s.reflowParameter.sn)

			// click query button
			query_button, _ := wd.FindElement(selenium.ByXPATH, "/html/body/div/div/div/div/section[2]/div[1]/div/div/div[2]/p/button")
			args := []interface{}{query_button}
			wd.ExecuteScript("arguments[0].click();", args)

			time.Sleep(time.Duration(3) * time.Second)

			html_parse, _ := wd.PageSource()
			doc := soup.HTMLParse(html_parse)
			table := doc.Find("div", "class", "react-bs-container-body").FindAll("tr")

			for i := 0; i < len(table); i++ {
				ele := "/html/body/div/div/div/div/section[2]/div[2]/div/div/div[2]/div[2]/div/div/div/div[2]/div[2]/table/tbody/tr[" + strconv.Itoa(i+1) + "]/td[1]"
				enter_modal, _ := wd.FindElement(selenium.ByXPATH, ele)
				enter_modal.Click()
				pop_up, _ := wd.WindowHandles()
				wd.SwitchWindow(pop_up[0])
				//fmt.Println(pop_up)
				time.Sleep(time.Duration(1) * time.Second)
				scrnshot, _ := wd.Screenshot()
				ioutil.WriteFile("test"+strconv.Itoa(i+1)+".png", scrnshot, 0666)
				modal, _ := wd.FindElement(selenium.ByClassName, "modal-body")
				loc, _ := modal.Location()
				sz, _ := modal.Size()
				// fmt.Println(loc)
				// fmt.Println(sz)
				file, _ := os.Open("./test" + strconv.Itoa(i+1) + ".png")
				defer file.Close()
				log.Info(file)
				img, _ := png.Decode(file)
				sub_image := img.(interface {
					SubImage(r image.Rectangle) image.Image
				}).SubImage(image.Rect(loc.X, loc.Y, loc.X+sz.Width, loc.Y+sz.Height))
				file, _ = os.Create("./crop" + strconv.Itoa(i+1) + ".png")
				png.Encode(file, sub_image)

				time.Sleep(time.Duration(5) * time.Second)

				// pdf_button, _ := wd.FindElement(selenium.ByXPATH, "/html/body/div[2]/div/div[2]/div/div/div[3]/div/div[2]/button[2]")
				// args := []interface{}{pdf_button}
				// wd.ExecuteScript("arguments[0].click();", args)
				// time.Sleep(time.Duration(5) * time.Second)

				ok_button, _ := wd.FindElement(selenium.ByXPATH, "/html/body/div[2]/div/div[2]/div/div/div[3]/div/div[2]/button[4]")
				args = []interface{}{ok_button}
				wd.ExecuteScript("arguments[0].click();", args)
				time.Sleep(time.Duration(2) * time.Second)
			}

			wd.Quit()
			service.Stop()

			////// 5. Get the result to the gin context ////////

			var images []string
			for i := 0; i < len(table); i++ {
				imgFile, _ := os.Open("./crop" + strconv.Itoa(i+1) + ".png")
				defer imgFile.Close()

				// create a new buffer base on file size
				fInfo, _ := imgFile.Stat()
				var size int64 = fInfo.Size()
				buf := make([]byte, size)
				// read file content into buffer
				fReader := bufio.NewReader(imgFile)
				fReader.Read(buf)
				imgBase64Str := base64.StdEncoding.EncodeToString(buf)

				images = append(images, imgBase64Str)
			}

			c.HTML(http.StatusOK, "img.html", gin.H{
				"images": images,
			})
		} else { // status code != 200
			log.Error("Record Not Found:" + strconv.Itoa(resp.StatusCode))
			c.JSON(http.StatusNotFound, gin.H{"error": "Record Not Found, the serial number doesn't exist"})
			c.Abort()
		}
	} else {
		log.Error("Failed to reach the destination", err)
		// responseCode(c, http.StatusInternalServerError, "Interal http server error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Interal http server error"})
		c.Abort()
	}
	resp.Body.Close()
}
