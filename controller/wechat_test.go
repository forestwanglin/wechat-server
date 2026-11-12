package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"wechat-server/common"
)

type wechatTestTransport func(*http.Request) (*http.Response, error)

func (f wechatTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWeChatCodeProfile(t *testing.T) {
	for _, tc := range []struct {
		name, body   string
		status       int
		profile      bool
		requestError bool
	}{
		{name: "profile", body: `{"openid":"user-id","nickname":"昵称\n内容","headimgurl":"https://example.com/avatar"}`, profile: true},
		{name: "nickname only", body: `{"openid":"user-id","nickname":"昵称"}`, profile: true},
		{name: "avatar only", body: `{"openid":"user-id","headimgurl":"https://example.com/avatar"}`, profile: true},
		{name: "redirect rejected", status: 302},
		{name: "missing fields", body: `{"openid":"user-id","subscribe":1}`},
		{name: "wechat error", body: `{"errcode":40001,"errmsg":"secret-token"}`},
		{name: "mismatched identity", body: `{"openid":"other-id","nickname":"wrong"}`},
		{name: "invalid JSON", body: `not json`},
		{name: "HTTP failure", status: 502},
		{name: "network failure", requestError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldTransport := http.DefaultTransport
			oldOut, oldErr := gin.DefaultWriter, gin.DefaultErrorWriter
			var logs bytes.Buffer
			gin.DefaultWriter, gin.DefaultErrorWriter = &logs, &logs
			t.Cleanup(func() {
				http.DefaultTransport = oldTransport
				gin.DefaultWriter, gin.DefaultErrorWriter = oldOut, oldErr
			})
			calls := 0
			http.DefaultTransport = wechatTestTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/cgi-bin/token" {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"secret-token","expires_in":7200}`)), Header: make(http.Header)}, nil
				}
				calls++
				if r.URL.Scheme != "https" || r.URL.Host != "api.weixin.qq.com" || r.URL.Path != "/cgi-bin/user/info" || r.URL.Query().Get("openid") != "user-id" || r.URL.Query().Get("access_token") != "secret-token" {
					t.Errorf("unexpected profile request destination or parameters")
				}
				if tc.requestError {
					return nil, fmt.Errorf("request failed: secret-token")
				}
				status := tc.status
				if status == 0 {
					status = 200
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})
			common.RefreshAccessToken()
			common.RegisterWeChatCodeAndID("test-code", "user-id")
			for attempt := 0; attempt < 2; attempt++ {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest("GET", "/api/wechat/user?code=test-code", nil)
				GetUserIDByCode(c)
				var result struct {
					Success bool               `json:"success"`
					Data    string             `json:"data"`
					Profile *wechatUserProfile `json:"profile"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if !result.Success {
					t.Fatal("legacy response success changed")
				}
				if attempt == 0 {
					if result.Data != "user-id" {
						t.Errorf("OpenID changed: %q", result.Data)
					}
					if (result.Profile != nil) != tc.profile {
						t.Errorf("profile presence: got %v want %v", result.Profile, tc.profile)
					}
					if result.Profile != nil && tc.name != "avatar only" && (result.Profile.Nickname != "昵称\n内容" && result.Profile.Nickname != "昵称") {
						t.Errorf("unexpected nickname: %q", result.Profile.Nickname)
					}
					if (tc.name == "profile" || tc.name == "avatar only") && result.Profile != nil && result.Profile.AvatarURL != "https://example.com/avatar" {
						t.Error("avatar not mapped")
					}
				} else if result.Data != "" || result.Profile != nil {
					t.Error("consumed code returned identity or profile")
				}
			}
			if calls != 1 {
				t.Errorf("profile requests: got %d want 1", calls)
			}
			if strings.Contains(logs.String(), "secret-token") || strings.Contains(logs.String(), "test-code") {
				t.Error("credentials leaked to logs")
			}
			if tc.profile && !strings.Contains(logs.String(), "nickname=") {
				t.Error("profile not logged")
			}
			if strings.Contains(logs.String(), "昵称\n内容") {
				t.Error("unescaped newline in profile log")
			}
		})
	}
}
