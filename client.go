package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type customTransport struct {
	http.Transport
}

func (t *customTransport) Dialer(network, addr string) (net.Conn, error) {
	conn, err := net.DialTimeout(network, addr, time.Second*15)
	if err != nil {
		fmt.Println("custom dialer error:", err)
		return nil, err
	}
	return conn, nil
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	//fmt.Println("Custom RoundTrip: ", req.URL)
	return t.Transport.RoundTrip(req)
}

type Booker struct {
	Client         *http.Client
	Date           string
	Username       string
	Password       string
	LoginData      string
	LaneToTrin     map[int]int
	Lane           int
	LaneId         int
	IdentifierData string
	Day            string
	Month          string
	STime          time.Time
	STime2         time.Time
	ETime          time.Time
	ETime2         time.Time
	TOD            string
	frmPmtRefNo    string
	frmClientID    string
	CSRF           string
	TGID           string
}

func (b *Booker) NewClient() {

	transport := &customTransport{}
	transport.Dial = transport.Dialer
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	b.Client = &http.Client{
		Jar:       jar,
		Transport: transport,
	}
}

func (b *Booker) GetInitialCookies() {
	req, err := http.NewRequest("GET", "https://clients.mindbodyonline.com/classic/ws?studioid=25730", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://clients.mindbodyonline.com/IdentityLogin/InitiateIdentityLogout")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="113", "Chromium";v="113", "Not-A.Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")

	resp, err := b.Client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
}

func (b *Booker) FormatLoginInfo() {
	year := time.Now().Year()

	/*
		file, err := os.Open("info.txt")
		if err != nil {
			panic(err)
		}

		defer file.Close()

		scanner := bufio.NewScanner(file)

		lineNumber := 0

		for scanner.Scan() {
			switch lineNumber {
			case 0:
				b.Username = url.QueryEscape(scanner.Text())
			case 1:
				b.Password = url.QueryEscape(scanner.Text())
			case 2:
				b.Lane, err = strconv.Atoi(url.QueryEscape(scanner.Text()))
				if err != nil {
					panic(err)
				}

				b.LaneId = b.LaneToTrin[b.Lane]
			case 3:
				RawSTime := url.QueryEscape(scanner.Text())
				if err != nil {
					panic(err)
				}

				b.STime, err = time.Parse("15", RawSTime)
				if err != nil {
					panic(err)
				}
				b.STime2 = b.STime.Add(30 * time.Minute)
				b.ETime = b.STime2
				b.ETime2 = b.STime2.Add(30 * time.Minute)
			case 4:
				b.TOD = scanner.Text()
			case 5:
				b.Month = scanner.Text()
			case 6:
				b.Day = scanner.Text()
			}
			lineNumber++
		} */

	b.Date = fmt.Sprintf(`%s/%s/%s`, b.Month, b.Day, fmt.Sprint(year))
	b.LoginData = fmt.Sprintf(`requiredtxtUserName=%s&requiredtxtPassword=%s&tg=&vt=&lvl=&stype=&qParam=&view=&trn=0&page=&catid=&prodid=&date=%s&classid=0&sSU=&optForwardingLink=&isAsync=false`, b.Username, b.Password, b.Date)
}

func (b *Booker) FormatLoginWebsite() {
	year := time.Now().Year()
	b.Date = fmt.Sprintf(`%s/%s/%s`, b.Month, b.Day, fmt.Sprint(year))
	b.LoginData = fmt.Sprintf(`requiredtxtUserName=%s&requiredtxtPassword=%s&tg=&vt=&lvl=&stype=&qParam=&view=&trn=0&page=&catid=&prodid=&date=%s&classid=0&sSU=&optForwardingLink=&isAsync=false`, b.Username, b.Password, b.Date)
}

func (b *Booker) PerformLogin() {
	//log.Println("Login Data:", (b.LoginData))

	req, err := http.NewRequest("POST", "https://clients.mindbodyonline.com/Login?studioID=25730&isLibAsync=true&isJson=true", strings.NewReader(b.LoginData))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://clients.mindbodyonline.com/classic/ws?studioid=25730")

	resp, err := b.Client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	log.Println("LOGIN BODY:", string(body))
}

func (b *Booker) PrepareBooking() {
	stime := b.STime.Format("15:04:05")
	etime := b.ETime.Format("15:04:05")

	laneOptions := map[int]bool{
		100000237: true,
		100000215: true,
		100000216: true,
		100000217: true,
	}

	if laneOptions[b.LaneId] {
		b.TGID = "11"

		endTimeObj, err := time.Parse("15:04:05", etime)
		if err != nil {
			panic(err)
		}
		endTimeObj = endTimeObj.Add(30 * time.Minute)
		etime = endTimeObj.Format("15:04:05")
	} else {
		b.TGID = "19"
	}

	baseURL := "https://clients.mindbodyonline.com/asp/appt_con.asp?loc=1&tgid=%s&trnid=%d&rtrnid=&date=%s&mask=False&STime=%s%s%s&ETime=%s%s%s"
	reqURL := fmt.Sprintf(baseURL, b.TGID, b.LaneId, b.Date, stime, "%20", b.TOD, etime, "%20", b.TOD)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		panic(err)
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		panic(err)
	}

	doc.Find("input[name='frmPmtRefNo']").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists {
			b.frmPmtRefNo = value
		}
	})

	doc.Find("input[name='frmClientID']").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists {
			b.frmClientID = value
		}
	})

	doc.Find("input[name='CSRFToken']").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists {
			b.CSRF = value
		}
	})
}

func (b *Booker) CompleteBooking() {
	stime := b.STime.Format("15:04:05") + " " + b.TOD
	etime := b.ETime.Format("15:04:05")
	blocklen := "30"
	visitType := "119"

	laneOptions := map[int]bool{
		100000237: true,
		100000215: true,
		100000216: true,
		100000217: true,
	}

	if laneOptions[b.LaneId] {
		b.TGID = "11"
		blocklen = "60"
		visitType = "136"

		endTimeObj, err := time.Parse("15:04:05", etime)
		if err != nil {
			panic(err)
		}
		endTimeObj = endTimeObj.Add(30 * time.Minute)
		etime = endTimeObj.Format("15:04:05")
	} else {
		b.TGID = "19"
	}

	baseURL := "https://clients.mindbodyonline.com/asp/adm/adm_appt_ap.asp?trnid=%d&rtrnid=&Date=%s&tgid=%s&tgBlockLength=%s&reSchedule=&origTrn=&origDate=&origId=&cType="
	reqURL := fmt.Sprintf(baseURL, b.LaneId, url.QueryEscape(b.Date), b.TGID, blocklen)

	baseCRF := "CSRFToken=%s&frmApptDate=%s&frmPmtRefNo=%s&reSchedule=&origId=&frmRtnAction=appt_con.asp%%3Floc%%3D1%%26tgid%%3D%s%%26trnid%%3D100000213%%26rtrnid%%3D100000213%%26date%%3D6%%2F6%%2F2023%%26clientid%%3D%%26Stime%%3D7%%3A00%%3A00+AM%%26Etime%%3D7%%3A30%%3A00+AM%%26rstime%%3D%%26retime%%3D%%26mask%%3DFalse%%26optResfor%%3D&frmRtnScreen=appt_con&frmProdVTID=119&frmUseXRegDB=0&frmXStudioID=&optReservedFor=&optPaidForOther=&OptSelf=&optLocation=1&optInstructor=%s&optVisitType=%s&frmClientID=%s&frmTrainerID=%s&tgCapacity=1&optStartTime=%s&optEndTime=%s&txtNotes=&Submit=Book+Appointment&name=https%%3A%%2F%%2Fclients.mindbodyonline.com%%2Fclassic%%2Fws%%3Fstudioid%%3D25730"
	CSRF := fmt.Sprintf(baseCRF, b.CSRF, url.QueryEscape(b.Date), b.frmPmtRefNo, b.TGID, fmt.Sprint(b.LaneId), visitType, b.frmClientID, fmt.Sprint(b.LaneId), url.QueryEscape(stime), url.QueryEscape(etime))

	var data = strings.NewReader(CSRF)

	req, err := http.NewRequest("POST", reqURL, data)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Host", "clients.mindbodyonline.com")
	req.Header.Set("Content-Length", "740")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Origin", "https://clients.mindbodyonline.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := b.Client.Do(req)
	if err != nil {
		fmt.Print("Err booking")
	}

	//body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	//fmt.Printf("Request Body: %s", body)
}
