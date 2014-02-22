package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/codegangsta/martini"
	"github.com/mbanzon/mailgun"
	"github.com/subosito/twilio"
)

var twilioAccount = os.Getenv("TWILIO_ACCOUNT")
var twilioKey = os.Getenv("TWILIO_KEY")
var mailgunPublicKey = os.Getenv("MAILGUN_PUBLIC_KEY")
var mailgunKey = os.Getenv("MAILGUN_KEY")
var smsFrom = os.Getenv("FROM_NUMBER")
var emailDomain = os.Getenv("FROM_DOMAIN")
var emailTo = os.Getenv("TO_EMAIL")

func main() {
	tc := twilio.NewClient(twilioAccount, twilioKey, nil)
	mc := mailgun.NewMailgun(emailDomain, mailgunKey, mailgunPublicKey)

	r := martini.NewRouter()
	m := martini.New()
	m.Use(martini.Logger())
	m.Use(martini.Recovery())
	m.Action(r.Handle)

	m.Map(tc)
	m.MapTo(mc, (*mailgun.Mailgun)(nil))

	r.Post("/sms", incomingSMS)
	r.Post("/email", incomingEmail)

	m.Run()
}

// self-pinger
// incoming call -> forward -> voicemail
// incoming voicemail -> email

func incomingCall() {
	// dial number
	// action=dialfinished
}

func dialFinished() {
	// check dialstatus
	// record transcribeAction=sendVoicemail
}

func sendVoicemail() {
	// send email
}

func incomingSMS(m mailgun.Mailgun, req *http.Request, log *log.Logger) {
	if err := verifyTwilioReq(req); err != nil {
		log.Print("Twilio request verification failed: ", err)
		return
	}

	log.Print("Got message from ", req.FormValue("From"))
	msg := mailgun.NewMessage(
		req.FormValue("From")+"@"+emailDomain,
		"New text from "+req.FormValue("From"),
		req.FormValue("Body"),
		emailTo,
	)
	msg.SetDKIM(true)
	_, _, err := m.Send(msg)
	if err != nil {
		log.Print("Email send error: ", err)
		return
	}
	log.Print("Email sent to ", emailTo)
}

func incomingEmail(tc *twilio.Client, req *http.Request, log *log.Logger) {
	if err := verifyMailgunSig(
		req.FormValue("token"),
		req.FormValue("timestamp"),
		req.FormValue("signature"),
	); err != nil {
		log.Print("Mailgun request verification failed: ", err)
		return
	}

	dkim := req.FormValue("X-Mailgun-Spf")
	spf := req.FormValue("X-Mailgun-Dkim-Check-Result")
	sender := req.FormValue("sender")
	if dkim != "Pass" || spf != "Pass" || sender != emailTo {
		log.Print("Email verification failed: SPF: %s, DKIM: %s, addr: %s", spf, dkim, sender)
		return
	}

	params := twilio.MessageParams{Body: req.FormValue("stripped-text")}
	dest := strings.SplitN(req.FormValue("recipient"), "@", 2)[0]
	_, _, err := tc.Messages.Send(smsFrom, dest, params)
	if err != nil {
		log.Print("SMS send failed: ", err)
		return
	}
	log.Print("SMS sent to ", dest)
}

func requestURL(req *http.Request) string {
	return "https://" + req.Host + req.RequestURI
}

func verifyTwilioReq(req *http.Request) error {
	req.ParseForm()
	return verifyTwilioSig(requestURL(req), req.PostForm, req.Header.Get("X-Twilio-Signature"))
}

func verifyTwilioSig(url string, data url.Values, signature string) error {
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}

	params := make([]string, 0, len(data))
	for k, vs := range data {
		for _, v := range vs {
			params = append(params, k+v)
		}
	}
	sort.Strings(params)

	h := hmac.New(sha1.New, []byte(twilioKey))
	h.Write([]byte(url + strings.Join(params, "")))
	if res := h.Sum(nil); !hmac.Equal(res, sig) {
		return fmt.Errorf("invalid signature: got %x, expected %x", res, sig)
	}
	return nil
}

func verifyMailgunSig(token, timestamp, signature string) error {
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return err
	}

	h := hmac.New(sha256.New, []byte(mailgunKey))
	h.Write([]byte(timestamp + token))
	if res := h.Sum(nil); !hmac.Equal(res, sig) {
		return fmt.Errorf("invalid signature: go %x, expected %x", res, sig)
	}
	return nil
}
