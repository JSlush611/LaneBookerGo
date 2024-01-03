package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
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
		return nil, fmt.Errorf("custom dialer error: %w", err)
	}
	return conn, nil
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
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

func (b *Booker) NewClient() error {
	transport := &customTransport{}
	transport.Dial = transport.Dialer

	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("error creating cookie jar: %w", err)
	}

	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	b.Client = &http.Client{
		Jar:       jar,
		Transport: transport,
	}
	return nil
}

func (b *Booker) GetInitialCookies() error {
	req, err := http.NewRequest("GET", "https://clients.mindbodyonline.com/classic/ws?studioid=25730", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
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
		return fmt.Errorf("error getting cookies: %w", err)
	}

	defer resp.Body.Close()
	return nil
}

func (b *Booker) FormatLoginWebsite() {
	year := time.Now().Year()
	b.Date = fmt.Sprintf(`%s/%s/%s`, b.Month, b.Day, fmt.Sprint(year))
	b.LoginData = fmt.Sprintf(`requiredtxtUserName=%s&requiredtxtPassword=%s&tg=&vt=&lvl=&stype=&qParam=&view=&trn=0&page=&catid=&prodid=&date=%s&classid=0&sSU=&optForwardingLink=&isAsync=false`, b.Username, b.Password, b.Date)
}

func (b *Booker) PerformLogin() error {
	req, err := http.NewRequest("POST", "https://clients.mindbodyonline.com/Login?studioID=25730&isLibAsync=true&isJson=true", strings.NewReader(b.LoginData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://clients.mindbodyonline.com/classic/ws?studioid=25730")

	resp, err := b.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error performing login: %w", err)
	}

	defer resp.Body.Close()
	//body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return fmt.Errorf("error performing login: %w", err)
	}

	//log.Println("LOGIN BODY:", string(body))
	return nil
}

func (b *Booker) PrepareBooking() error {
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
			return fmt.Errorf("error preparing booking: %w", err)
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
		return fmt.Errorf("error creating request: %w", err)
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("error parsing body: %w", err)
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

	return nil
}

func (b *Booker) CompleteBooking() error {
	stimeFormat := "3:04:05 PM"
	stime := b.STime.Format(stimeFormat)
	etime := b.ETime.Format(stimeFormat)

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
		endTimeObj := b.ETime.Add(30 * time.Minute)
		etime = endTimeObj.Format(stimeFormat)
	} else {
		b.TGID = "19"
	}

	reqURL := fmt.Sprintf("https://clients.mindbodyonline.com/asp/adm/adm_appt_ap.asp?trnid=%d&rtrnid=&Date=%s&tgid=%s&tgBlockLength=%s&reSchedule=&origTrn=&origDate=&origId=&cType=", b.LaneId, url.QueryEscape(b.Date), b.TGID, blocklen)

	CSRF := fmt.Sprintf("CSRFToken=%s&frmApptDate=%s&frmPmtRefNo=%s&reSchedule=&origId=&frmRtnAction=appt_con.asp%%3Floc%%3D1%%26tgid%%3D%s%%26trnid%%3D%d%%26rtrnid%%3D%d%%26date%%3D%s%%26clientid%%3D%%26Stime%%3D%s%%26Etime%%3D%s%%26rstime%%3D%%26retime%%3D%%26mask%%3DFalse%%26optResfor%%3D&frmRtnScreen=appt_con&frmProdVTID=%s&frmUseXRegDB=0&frmXStudioID=&optReservedFor=&optPaidForOther=&OptSelf=&optLocation=1&optInstructor=%d&optVisitType=%s&frmClientID=%s&frmTrainerID=%d&tgCapacity=1&optStartTime=%s&optEndTime=%s&txtNotes=&Submit=Book+Appointment&name=https%%3A%%2F%%2Fclients.mindbodyonline.com%%2Fclassic%%2Fws%%3Fstudioid%%3D25730", b.CSRF, url.QueryEscape(b.Date), b.frmPmtRefNo, b.TGID, b.LaneId, b.LaneId, url.QueryEscape(b.Date), url.QueryEscape(stime), url.QueryEscape(etime), visitType, b.LaneId, visitType, b.frmClientID, b.LaneId, url.QueryEscape(stime), url.QueryEscape(etime))

	var data = strings.NewReader(CSRF)

	req, err := http.NewRequest("POST", reqURL, data)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Host", "clients.mindbodyonline.com")
	req.Header.Set("Content-Length", strconv.Itoa(data.Len()))
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Origin", "https://clients.mindbodyonline.com")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := b.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	//     panic(err)
	// }
	// fmt.Printf("Request Body: %s", body)

	return nil
}
