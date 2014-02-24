package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

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
var callDest = os.Getenv("CALL_DEST")

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

	r.Post("/call", verifyTwilioReq, incomingCall)
	r.Post("/record-voicemail", verifyTwilioReq, recordVoicemail)
	r.Post("/voicemail", verifyTwilioReq, incomingVoicemail)
	r.Post("/sms", verifyTwilioReq, incomingSMS)
	r.Post("/email", incomingEmail)
	r.Post("/hangup", hangup)
	r.Get("/ping", func() {})

	go pinger()
	m.Run()
}

func pinger() {
	for _ = range time.Tick(time.Minute) {
		http.Get(os.Getenv("BASE_URL") + "/ping")
	}
}

func twilioResponse(s string) string {
	return xml.Header + "<Response>\n" + s + "\n</Response>"
}

func incomingCall() string {
	return twilioResponse(`<Dial timeout="5" action="/record-voicemail">` + callDest + `</Dial>`)
}

func recordVoicemail(req *http.Request) string {
	if req.FormValue("DialCallStatus") == "completed" {
		return hangup()
	}
	sayNumber := strings.Replace(strings.TrimLeft(smsFrom, "+1"), "", " ", -1)
	say := "<Say>You have reached" + sayNumber + ". Please leave a message after the tone.</Say>\n"
	return twilioResponse(say + `<Record action="/hangup" transcribeCallback="/voicemail" maxLength="120" />`)
}

func hangup() string {
	return twilioResponse("<Hangup/>")
}

type transcription struct {
	Duration int    `json:"duration,string"`
	Text     string `json:"transcription_text"`
}

var voicemailTemplate = template.Must(template.New("vm").Parse(`From: {{.From}}
Duration: {{.Duration}}s
Recording: {{.Recording}}

{{.Text}}`))

type voicemailData struct {
	From      string
	Duration  int
	Recording string
	Text      string
}

func incomingVoicemail(tc *twilio.Client, m mailgun.Mailgun, req *http.Request, log *log.Logger) {
	log.Printf("%#v", req.Form)
	transReq, err := tc.NewRequest("GET", req.FormValue("TranscriptionUrl")+".json", nil)
	if err != nil {
		log.Println("Transcription req build error:", err)
		return
	}
	var trans transcription
	_, err = tc.Do(transReq, &trans)
	if err != nil {
		log.Println("Transcription req error:", err)
		return
	}

	var buf bytes.Buffer
	err = voicemailTemplate.Execute(&buf, &voicemailData{
		From:      req.FormValue("From"),
		Duration:  trans.Duration,
		Recording: req.FormValue("RecordingUrl"),
		Text:      trans.Text,
	})
	if err != nil {
		log.Println("Email template error:", err)
		return
	}

	msg := mailgun.NewMessage("voicemail@"+emailDomain, "New voicemail from "+req.FormValue("From"), buf.String(), emailTo)
	msg.SetDKIM(true)
	_, _, err = m.Send(msg)
	if err != nil {
		log.Println("Voicemail send error:", err)
		return
	}
}

func incomingSMS(m mailgun.Mailgun, req *http.Request, log *log.Logger) {
	log.Println(req.Form)
	msg := mailgun.NewMessage(
		req.FormValue("From")+"@"+emailDomain,
		"New text from "+req.FormValue("From"),
		req.FormValue("Body"),
		emailTo,
	)
	msg.SetDKIM(true)
	_, _, err := m.Send(msg)
	if err != nil {
		log.Println("Email send error:", err)
		return
	}
	log.Println("Email sent to", emailTo)
}

func incomingEmail(tc *twilio.Client, req *http.Request, log *log.Logger) {
	if err := verifyMailgunSig(
		req.FormValue("token"),
		req.FormValue("timestamp"),
		req.FormValue("signature"),
	); err != nil {
		log.Println("Mailgun request verification failed:", err)
		return
	}

	dkim := req.FormValue("X-Mailgun-Spf")
	spf := req.FormValue("X-Mailgun-Dkim-Check-Result")
	sender := req.FormValue("sender")
	if dkim != "Pass" || spf != "Pass" || sender != emailTo {
		log.Printf("Email verification failed: SPF: %s, DKIM: %s, addr: %s", spf, dkim, sender)
		return
	}

	params := twilio.MessageParams{Body: req.FormValue("stripped-text")}
	dest := strings.SplitN(req.FormValue("recipient"), "@", 2)[0]
	_, _, err := tc.Messages.Send(smsFrom, dest, params)
	if err != nil {
		log.Println("SMS send failed:", err)
		return
	}
	log.Println("SMS sent to", dest)
}

func requestURL(req *http.Request) string {
	return "https://" + req.Host + req.RequestURI
}

func verifyTwilioReq(w http.ResponseWriter, req *http.Request, log *log.Logger) {
	req.ParseForm()
	err := verifyTwilioSig(requestURL(req), req.PostForm, req.Header.Get("X-Twilio-Signature"))
	if err != nil {
		log.Println("Twilio request verification failed:", err)
		w.WriteHeader(403)
		return
	}
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
