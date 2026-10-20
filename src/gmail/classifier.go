package gmail

import (
	"bytes"
	"encoding/base64"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	categoryPersonal   = "CATEGORY_PERSONAL"
	categoryPromotions = "CATEGORY_PROMOTIONS"
	categorySocial     = "CATEGORY_SOCIAL"
	categoryUpdates    = "CATEGORY_UPDATES"
	maxClassifierBody  = 64 * 1024
)

var socialDomains = []string{
	"discord.com",
	"facebook.com",
	"facebookmail.com",
	"instagram.com",
	"linkedin.com",
	"nextdoor.com",
	"pinterest.com",
	"reddit.com",
	"redditmail.com",
	"slack.com",
	"snapchat.com",
	"tiktok.com",
	"twitter.com",
	"x.com",
}

var socialPhrases = []string{
	"friend request",
	"mentioned you",
	"new comment on",
	"new follower",
	"started following",
	"tagged you",
	"invited you to join",
	"liked your",
	"replied to your",
	"shared a post",
	"new connection",
}

var updatePhrases = []string{
	"account alert",
	"account verification",
	"appointment reminder",
	"delivery update",
	"email verification",
	"invoice",
	"login code",
	"one-time code",
	"order confirmation",
	"order received",
	"order shipped",
	"password reset",
	"payment received",
	"receipt",
	"reservation confirmed",
	"security alert",
	"shipment update",
	"shipping confirmation",
	"statement ready",
	"tracking number",
	"verification code",
}

var promotionPhrases = []string{
	"clearance",
	"coupon",
	"discount",
	"limited time",
	"promo code",
	"special offer",
	"shop now",
	"save up to",
	"sale ends",
	"weekend sale",
}

// classifyEmail chooses a Gmail category from common sender, header, subject,
// and body signals. Messages that do not match a stronger signal go to Primary.
func classifyEmail(raw []byte) string {
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return categoryPersonal
	}

	subject := message.Header.Get("Subject")
	if decoded, err := new(mime.WordDecoder).DecodeHeader(subject); err == nil {
		subject = decoded
	}

	fromDomain := ""
	if from, err := mail.ParseAddress(message.Header.Get("From")); err == nil {
		if separator := strings.LastIndex(from.Address, "@"); separator >= 0 {
			fromDomain = strings.ToLower(from.Address[separator+1:])
		}
	}

	contentType := message.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}
	used := 0
	body := readClassifierBody(
		io.LimitReader(message.Body, maxClassifierBody),
		contentType,
		message.Header.Get("Content-Transfer-Encoding"),
		&used,
		0,
	)
	text := strings.ToLower(strings.Join([]string{subject, body}, "\n"))

	if containsAny(text, socialPhrases) {
		return categorySocial
	}
	if containsAny(text, updatePhrases) {
		return categoryUpdates
	}
	if message.Header.Get("List-Unsubscribe") != "" || containsAny(text, promotionPhrases) || containsPromoWord(text) {
		return categoryPromotions
	}
	if isSocialDomain(fromDomain) {
		return categorySocial
	}
	return categoryPersonal
}

func readClassifierBody(body io.Reader, contentType, transferEncoding string, used *int, depth int) string {
	if depth > 5 || *used >= maxClassifierBody {
		return ""
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType == "" {
		mediaType = "text/plain"
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return ""
		}
		reader := multipart.NewReader(body, boundary)
		var text strings.Builder
		for *used < maxClassifierBody {
			part, err := reader.NextRawPart()
			if err != nil {
				break
			}
			partText := readClassifierBody(part, part.Header.Get("Content-Type"), part.Header.Get("Content-Transfer-Encoding"), used, depth+1)
			if partText != "" {
				text.WriteString(partText)
				text.WriteByte('\n')
			}
			if err := part.Close(); err != nil {
				break
			}
		}
		return text.String()
	}

	var decoded io.Reader = body
	switch strings.ToLower(transferEncoding) {
	case "base64":
		decoded = base64.NewDecoder(base64.StdEncoding, body)
	case "quoted-printable":
		decoded = quotedprintable.NewReader(body)
	}

	remaining := maxClassifierBody - *used
	data, _ := io.ReadAll(io.LimitReader(decoded, int64(remaining)))
	*used += len(data)
	if !utf8.Valid(data) {
		data = bytes.ToValidUTF8(data, []byte(" "))
	}

	switch {
	case mediaType == "text/plain":
		return string(data)
	case mediaType == "text/html":
		return stripHTML(string(data))
	default:
		return ""
	}
}

func containsAny(text string, phrases []string) bool {
	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func containsPromoWord(text string) bool {
	for _, word := range strings.FieldsFunc(text, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		switch word {
		case "deal", "deals", "offer", "offers", "promotion", "promotions", "sale", "sales":
			return true
		}
	}
	return false
}

func isSocialDomain(domain string) bool {
	for _, socialDomain := range socialDomains {
		if domain == socialDomain || strings.HasSuffix(domain, "."+socialDomain) {
			return true
		}
	}
	return false
}

func stripHTML(text string) string {
	var stripped strings.Builder
	inTag := false
	for _, r := range text {
		switch r {
		case '<':
			inTag = true
			stripped.WriteByte(' ')
		case '>':
			inTag = false
		default:
			if !inTag {
				stripped.WriteRune(r)
			}
		}
	}
	return strings.ToLower(html.UnescapeString(stripped.String()))
}
