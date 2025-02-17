package modules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"server-go/common"

	"golang.org/x/oauth2"
)

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
}

var oauthEndpoint = oauth2.Endpoint{
	AuthURL:   common.Config.ApiEndpoint + "/oauth2/authorize",
	TokenURL:  common.Config.ApiEndpoint + "/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

type InteractionResponse struct {
	Type int `json:"type"` // 1 = Pong ,4 = Respond
	Data struct {
		Content string  `json:"content"`
		Embeds  []Embed `json:"embeds"`
	} `json:"data"`
}

type EmbedFooter struct {
	Text         string `json:"text"`
	IconURL      string `json:"icon_url"`
	ProxyIconURL string `json:"proxy_icon_url"`
}

type InteractionsData struct {
	Type int `json:"type"` // 1 = ping
	Data struct {
		ID string `json:"custom_id"`
	}
	Message struct {
		Content string `json:"content"`
	}

	Member struct {
		User struct {
			ID            string `json:"id"`
			Username      string `json:"username"`
			Discriminator string `json:"discriminator"`
		} `json:"user"`
	} `json:"member"`
}

func Interactions(data InteractionsData) (string, error) {
	if data.Type == 1 {
		return "{\"type\":1}", nil //copilot I hope you die
	}

	response := InteractionResponse{}

	response.Type = 4

	userid, _ := strconv.ParseInt(data.Member.User.ID, 10, 64)

	action := strings.Split(data.Data.ID, ":")

	if data.Type == 3 && IsUserAdminDC(userid) {

		response.Data.Embeds = []Embed{{
			Footer: EmbedFooter{
				Text: fmt.Sprintf("Admin: %s#%s (%s)", data.Member.User.Username, data.Member.User.Discriminator, data.Member.User.ID),
			},
		}}

		firstVariable, _ := strconv.ParseInt(action[1], 10, 32) // if action is delete review or delete_and_ban its reviewid otherwise userid
		if action[0] == "delete_review" {
			err := DeleteReview(int32(firstVariable), common.Config.AdminToken)
			if err == nil {
				response.Data.Content = "Successfully Deleted review with id " + action[1]
			} else {
				response.Data.Content = err.Error()
			}
		} else if action[0] == "ban_user" {
			err := BanUser(action[1], common.Config.AdminToken)
			if err == nil {
				response.Data.Content = "Successfully banned user with id " + action[1]
			} else {
				response.Data.Content = err.Error()
			}

		} else if action[0] == "delete_and_ban" {
			err := DeleteReview(int32(firstVariable), common.Config.AdminToken)
			err2 := BanUser(action[2], common.Config.AdminToken)
			if err == nil && err2 == nil {
				response.Data.Content = "Successfully Deleted review with id " + action[1] + " and banned user with id " + action[2]
			} else {
				response.Data.Content = err.Error() + err2.Error() // I hope this doesnt create error
			}
		}
	}
	if response.Data.Content != "" {
		b, err := json.Marshal(response)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", errors.New("invalid interaction")
}

func ExchangeCodePlus(code, redirectURL string) (string, error) {
	conf := &oauth2.Config{
		Endpoint:     oauthEndpoint,
		Scopes:       []string{"identify"},
		RedirectURL:  redirectURL,
		ClientID:     common.Config.ClientId,
		ClientSecret: common.Config.ClientSecret,
	}

	token, err := conf.Exchange(context.Background(), code)
	if err != nil {
		return "", err
	} else {
		return token.AccessToken, nil
	}

}

func GetUser(token string) (user *DiscordUser, err error) {
	// TODO discordid is always 0 fix
	req, _ := http.NewRequest(http.MethodGet, common.Config.ApiEndpoint+"/users/@me", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return user, nil
	}
	return nil, err
}

func GetUserViaID(userid int64) (user *DiscordUser, err error) {
	req, _ := http.NewRequest(http.MethodGet, common.Config.ApiEndpoint+"/users/"+fmt.Sprint(userid), nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+common.Config.BotToken)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		err = json.NewDecoder(resp.Body).Decode(&user)
		resp.Body.Close()
		return user, nil
	}
	return nil, err
}

func ExchangeCode(token string) (string, error) {
	return ExchangeCodePlus(token, common.Config.RedirectUri)
}

type ReportWebhookEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Embed struct {
	Fields []ReportWebhookEmbedField `json:"fields"`
	Footer EmbedFooter               `json:"footer"`
}

type WebhookEmoji struct {
	Name     string `json:"name,omitempty"`
	ID       string `json:"id,omitempty"`
	Animated bool   `json:"animated,omitempty"`
}

type WebhookComponent struct {
	Type       int                `json:"type"`
	Style      int                `json:"style"`
	Label      string             `json:"label"`
	CustomID   string             `json:"custom_id"`
	Emoji      WebhookEmoji       `json:"emoji,omitempty"`
	Components []WebhookComponent `json:"components"`
}

type ReportWebhookData struct {
	Content    string             `json:"content"`
	Username   string             `json:"username"`
	AvatarURL  string             `json:"avatar_url"`
	Embeds     []Embed            `json:"embeds"`
	Components []WebhookComponent `json:"components"`
}

func SendReportWebhook(data ReportWebhookData) error {
	body, err := json.Marshal(data)
	var resp *http.Response

	resp, err = http.Post(common.Config.DiscordWebhook, "application/json", strings.NewReader(string(body)))
	bodyBytes, err := io.ReadAll(resp.Body)
	print(string(bodyBytes))
	return err
}

func GetProfilePhotoURL(userid string, avatar string) string {
	if strings.HasPrefix(avatar, "a_") {
		return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.gif", userid, avatar)
	} else {
		return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userid, avatar)
	}
}

type Snowflake uint64

func (s *Snowflake) UnmarshalJSON(v []byte) error {
	parsed, err := strconv.ParseUint(strings.Trim(string(v), `"`), 10, 64)
	if err != nil {
		return err
	}

	*s = Snowflake(parsed)
	return nil
}
