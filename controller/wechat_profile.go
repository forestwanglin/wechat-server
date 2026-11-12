package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"wechat-server/common"
)

type wechatUserProfile struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

// Profile lookup is optional: it must never invalidate a verified login.
func getWeChatUserProfile(openID string) *wechatUserProfile {
	token := common.GetAccessToken()
	if token == "" {
		common.SysError("WeChat profile unavailable: access token is not configured")
		return nil
	}
	query := url.Values{"access_token": {token}, "openid": {openID}, "lang": {"zh_CN"}}
	client := http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Get("https://api.weixin.qq.com/cgi-bin/user/info?" + query.Encode())
	if err != nil {
		// HTTP errors may contain the request URL and access token.
		common.SysError("WeChat profile lookup failed: request error")
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		common.SysError(fmt.Sprintf("WeChat profile lookup failed: HTTP %d", res.StatusCode))
		return nil
	}
	var info struct {
		wechatResponse
		OpenID     string `json:"openid"`
		Nickname   string `json:"nickname"`
		HeadImgURL string `json:"headimgurl"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 64*1024)).Decode(&info); err != nil {
		common.SysError("WeChat profile lookup failed: invalid response")
		return nil
	}
	if info.ErrCode != 0 {
		common.SysError(fmt.Sprintf("WeChat profile lookup failed: errcode=%d", info.ErrCode))
		return nil
	}
	if info.OpenID != openID {
		common.SysError("WeChat profile lookup failed: OpenID mismatch")
		return nil
	}
	if info.Nickname == "" && info.HeadImgURL == "" {
		common.SysLog("WeChat profile unavailable: nickname and avatar are empty")
		return nil
	}
	common.SysLog(fmt.Sprintf("WeChat profile received: nickname=%q avatar_url=%q", info.Nickname, info.HeadImgURL))
	return &wechatUserProfile{Nickname: info.Nickname, AvatarURL: info.HeadImgURL}
}
