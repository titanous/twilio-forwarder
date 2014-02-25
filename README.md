# Twilio Forwarder

This is a tiny Go app that forwards phone calls, records voicemail, and bridges
SMS to email. You'll need a paid [Twilio](https://www.twilio.com/) account and
free [Mailgun](http://www.mailgun.com/) account to get up and running.

It runs on Heroku with the [Go
buildpack](https://github.com/kr/heroku-buildpack-go).

## Environment Variables

Name | Description | Example
---- | ----------- | -------
`TWILIO_ACCOUNT` | Twilio Account Sid | `AC12345678123456781234567812345678`
`TWILIO_KEY` | Twilio Auth Token | `f1d2d2f924e986ac86fdf7b36c94bcdf32beec15`
`MAILGUN_PUBLIC_KEY` | Mailgun Public Key | `pubkey-3ax6xnjp29jd6fds4gc373sgvjxteol0`
`MAILGUN_KEY` | Mailgun API key | `key-3ax6xnjp29jd6fds4gc373sgvjxteol0`
`FROM_NUMBER` | Source number for outgoing SMS messages. Must be confirmed or owned in Twilio. | `+15555551212`
`FROM_DOMAIN` | Domain to use for email. Must be configured in Mailgun | `sms.example.com`
`TO_EMAIL` | Email to send/receive SMS messages and voicemails to. Must sign outbound messages with DKIM and have valid SPF. | `user@example.com`
`CALL_DEST` | Number to forward phone calls to. Should not have voicemail enabled. | `+15555551212`
`BASE_URL` | The URL to the application with no trailing slash | `https://example.herokuapp.com`

## Setup

**Twilio**

- Set *Voice Request URL* to `$BASE_URL/call`.
- Set *Messaging Request URL* to `$BASE_URL/sms`.

**Mailgun**

- Add a route:
  - *Filter Expression:* `catch_all()`
  - *Action:* `forward("$BASE_URL/email")`

## Usage

Send emails to `$DEST@$FROM_DOMAIN` (where `$DEST` is something like
`+15555551212`) from `$TO_EMAIL` to send outbound SMS messages. Inbound SMS
messages will be sent to `$TO_EMAIL`. Reply to SMS emails to send outbound
replies. Inbound calls will be forwarded to `$CALL_DEST`. If the call is not
answered a voicemail message will be recorded and the recording and transcript
will be emailed to `$TO_EMAIL`.

This is obviously just a start and could be modified to be more sophisticated.
Pull requests welcome!
