package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/corpix/uarand"
	"golang.org/x/exp/rand"
)

const (
	maxRetries           = 4
	successLoginPattern  = `{"success":true,"json":{"success":true`
	failureLoginPattern1 = `{"success":true,"json":{"success":false`
	failureLoginPattern2 = `{"success":false,"sessionExpired":true}`
)

// isSuccessfulResponse checks if the response contains the success indicator
func isSuccessfulResponse(resp *http.Response) (bool, error) {
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// Reset the body for future reads
	resp.Body = io.NopCloser(strings.NewReader(string(body)))

	// Check the body content for success indication
	if strings.Contains(string(body), `window.location = "../my_sch.asp`) && !strings.Contains(string(body), `The appointment was not booked.`) {
		return true, nil
	}

	return false, nil
}

type customTransport struct {
	http.Transport
}

func (t *customTransport) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
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
	Client           *http.Client
	Date             string
	Username         string
	Password         string
	LoginData        string
	LaneToTrin       map[int]int
	Lane             int
	LaneId           int
	IdentifierData   string
	Day              string
	Month            string
	RawSTime         string
	STime            time.Time
	STime2           time.Time
	ETime            time.Time
	ETime2           time.Time
	TOD              string
	frmPmtRefNo      string
	frmClientID      string
	CSRF             string
	TGID             string
	halfHourSelected bool
	UserAgent        string
}

func (b *Booker) NewClient() error {
	transport := &customTransport{
		http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
				CipherSuites: shuffleCipherSuites([]uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				}),
			},
		},
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("error creating cookie jar: %w", err)
	}

	b.Client = &http.Client{
		Jar:       jar,
		Transport: transport,
	}

	b.UserAgent = uarand.GetRandom()

	return nil
}

// Function to shuffle cipher suites
func shuffleCipherSuites(cipherSuites []uint16) []uint16 {
	rand.Seed(uint64(time.Now().UnixNano()))
	rand.Shuffle(len(cipherSuites), func(i, j int) {
		cipherSuites[i], cipherSuites[j] = cipherSuites[j], cipherSuites[i]
	})
	return cipherSuites
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
	req.Header.Set("User-Agent", b.UserAgent)

	fmt.Println("Sending Request Headers:")
	for key, value := range req.Header {
		fmt.Printf("%s: %s\n", key, value)
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting cookies: %w", err)
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Headers:")
	for key, value := range resp.Header {
		fmt.Printf("%s: %s\n", key, value)
	}

	cookies := resp.Cookies()
	fmt.Println("Cookies from Response:")
	for _, cookie := range cookies {
		fmt.Printf("%s: %s\n", cookie.Name, cookie.Value)
	}

	u, err := url.Parse("https://clients.mindbodyonline.com")
	if err != nil {
		return fmt.Errorf("error parsing URL: %w", err)
	}

	allCookies := b.Client.Jar.Cookies(u)
	fmt.Println("All Cookies in Jar After Request:")
	for _, cookie := range allCookies {
		fmt.Printf("%s: %s\n", cookie.Name, cookie.Value)
	}

	return nil
}

func (b *Booker) FormatLoginWebsite() {
	year := time.Now().Year()
	b.Date = fmt.Sprintf(`%s/%s/%d`, b.Month, b.Day, year)
	b.LoginData = fmt.Sprintf(`requiredtxtUserName=%s&requiredtxtPassword=%s&tg=&vt=&lvl=&stype=&qParam=&view=&trn=0&page=&catid=&prodid=&date=%s&classid=0&sSU=&optForwardingLink=&isAsync=false`, b.Username, b.Password, b.Date)
}

func (b *Booker) PerformLogin() error {
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequest("POST", "https://clients.mindbodyonline.com/Login?studioID=25730&isLibAsync=true&isJson=true", strings.NewReader(b.LoginData))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Referer", "https://clients.mindbodyonline.com/classic/ws?studioid=25730")
		req.Header.Set("User-Agent", b.UserAgent)

		resp, err := b.Client.Do(req)
		if err != nil {
			log.Printf("Error performing login: %v. Retrying... (%d/%d)", err, i+1, maxRetries)
			b.NewClient()

			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %w", err)
		}

		log.Println("Login body:", string(body))

		// Check if login was successful or if the session expired
		if strings.Contains(string(body), successLoginPattern) {
			return nil
		} else if strings.Contains(string(body), failureLoginPattern1) || strings.Contains(string(body), failureLoginPattern2) {
			log.Printf("Login failed or session expired. Retrying... (%d/%d)", i+1, maxRetries)

			b.NewClient()

			continue
		}

		log.Printf("Unexpected login response. Retrying... (%d/%d)", i+1, maxRetries)
		b.NewClient()
	}

	return fmt.Errorf("login failed after %d attempts", maxRetries)
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

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://clients.mindbodyonline.com/classic/ws?studioid=25730")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="113", "Chromium";v="113", "Not-A.Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", b.UserAgent)

	resp, err := b.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
	for i := 0; i < maxRetries; i++ {
		stimeFormat := "3:04:05 PM"

		if b.TOD == "PM" {
			if b.STime.Hour() < 12 {
				b.STime = b.STime.Add(time.Hour * 12)
			}

			if b.ETime.Hour() < 12 {
				b.ETime = b.ETime.Add(time.Hour * 12)
			}
		}

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
		req.Header.Set("User-Agent", b.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := b.Client.Do(req)
		if err != nil {
			log.Printf("Error sending booking request: %v. Retrying... (%d/%d)", err, i+1, maxRetries)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v. Retrying... (%d/%d)", err, i+1, maxRetries)
			continue
		}

		successful, err := isSuccessfulResponse(resp)
		if err != nil {
			return fmt.Errorf("error checking booking response: %w", err)
		}

		if successful {
			log.Printf("Booking successful. Request Body: %s", string(body)[:200]) // Print first 200 characters
			return nil
		}

		log.Printf("Booking not successful. Retrying... (%d/%d)", i+1, maxRetries)
	}

	log.Printf("booking failed after %d attempts", maxRetries)
	return fmt.Errorf("booking failed after %d attempts", maxRetries)
}
