package gmail

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyEmail(t *testing.T) {
	testCases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "defaults to primary",
			raw:  "From: person@example.com\r\nSubject: Hello\r\n\r\nHi there",
			want: categoryPersonal,
		},
		{
			name: "classifies social notification",
			raw:  "From: notify@example.com\r\nSubject: Alex mentioned you\r\n\r\nYou were mentioned in a post.",
			want: categorySocial,
		},
		{
			name: "recognizes social sender domain",
			raw:  "From: updates@linkedin.com\r\nSubject: Your network\r\n\r\nSee what's new.",
			want: categorySocial,
		},
		{
			name: "classifies mailing list promotion",
			raw:  "From: shop@example.com\r\nSubject: A note\r\nList-Unsubscribe: <mailto:unsubscribe@example.com>\r\n\r\nHello",
			want: categoryPromotions,
		},
		{
			name: "classifies promotion from body text",
			raw:  "From: shop@example.com\r\nSubject: A note\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nGet a special=20offer today!",
			want: categoryPromotions,
		},
		{
			name: "classifies updates from multipart body",
			raw: "From: store@example.com\r\nSubject: Your package\r\nContent-Type: multipart/alternative; boundary=msg\r\n\r\n" +
				"--msg\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nYour order shipped. Tracking number: 123.\r\n" +
				"--msg\r\nContent-Type: text/html\r\n\r\n<p>Your order shipped</p>\r\n--msg--\r\n",
			want: categoryUpdates,
		},
		{
			name: "social notification takes precedence over promotional signals",
			raw:  "From: notify@example.com\r\nSubject: New follower\r\nList-Unsubscribe: <mailto:unsubscribe@example.com>\r\n\r\nA new follower started following you.",
			want: categorySocial,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, classifyEmail([]byte(tc.raw)))
		})
	}
}

func TestReadClassifierBodyLimitsBytes(t *testing.T) {
	body := strings.Repeat("x", maxClassifierBody+100)
	used := 0

	got := readClassifierBody(strings.NewReader(body), "text/plain", "", &used, 0)

	assert.Len(t, got, maxClassifierBody)
	assert.Equal(t, maxClassifierBody, used)
}
