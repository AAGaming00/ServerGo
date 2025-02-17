package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"server-go/common"
	"server-go/database"
	"server-go/modules"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

type Response struct {
	Successful bool   `json:"successful"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

type ReviewDBAuthResponse struct {
	Successful bool   `json:"successful"`
	Message    string `json:"message"`
	Token      string `json:"token"`
}

var AddUserReview = func(w http.ResponseWriter, r *http.Request) {
	response := Response{}

	var data modules.UR_RequestData
	json.NewDecoder(r.Body).Decode(&data)

	if len(data.Comment) > 1000 {
		response.Message = "Comment Too Long"
	} else if len(strings.TrimSpace(data.Comment)) == 0 {
		response.Message = "Write Something Guh"
	}

	if slices.Contains(common.OptedOut, uint64(data.DiscordID)) {
		response.Message = "This user opted out"
	}

	if response.Message != "" {
		common.SendStructResponse(w, response)
		return
	}

	res, err := modules.AddReview(data.DiscordID, data.Token, data.Comment, int32(data.ReviewType))
	if err != nil {
		response.Successful = false
		response.Message = "An error occurred"
		println(err.Error())
	} else {
		response.Successful = true
		response.Message = res
	}

	common.SendStructResponse(w, response)
}

var ClientMods []string = []string{"aliucord", "betterdiscord", "powercordv2", "replugged", "enmity", "vencord", "vendetta"}

var ReviewDBAuth = func(w http.ResponseWriter, r *http.Request) {
	clientmod := r.URL.Query().Get("clientMod")
	if clientmod == "" {
		clientmod = "aliucord"
	}

	if !slices.Contains(ClientMods, clientmod) {
		common.SendStructResponse(w, ReviewDBAuthResponse{
			Successful: false,
			Message:    "Invalid clientMod",
		})
		return
	}

	token, err := modules.AddUserReviewsUser(r.URL.Query().Get("code"), clientmod)

	if err != nil {
		io.WriteString(w, `{"token": "", "successful": false}`)
		return
	}

	res := ReviewDBAuthResponse{
		Token:      token,
		Successful: true,
	}

	response, _ := json.Marshal(res)
	io.WriteString(w, string(response))
}

var ReportReview = func(w http.ResponseWriter, r *http.Request) {
	var data modules.ReportData
	json.NewDecoder(r.Body).Decode(&data)

	response := Response{}

	if data.Token == "" || data.ReviewID == 0 {
		response.Message = "Invalid Request"
		common.SendStructResponse(w, response)
		return
	}

	err := modules.ReportReview(data.ReviewID, data.Token)
	if err != nil {
		response.Message = err.Error()
		common.SendStructResponse(w, response)
		return
	}
	response.Successful = true
	response.Message = "Successfully Reported Review"
	common.SendStructResponse(w, response)
}

var DeleteReview = func(w http.ResponseWriter, r *http.Request) {
	var data modules.ReportData //both reportdata and deletedata are same
	json.NewDecoder(r.Body).Decode(&data)

	responseData := Response{
		Successful: false,
		Message:    "",
	}

	if data.Token == "" || data.ReviewID == 0 {
		responseData.Message = "Invalid Request"
		res, _ := json.Marshal(responseData)

		w.Write(res)
		return
	}

	err := modules.DeleteReview(data.ReviewID, data.Token)
	if err != nil {
		responseData.Message = err.Error()
		res, _ := json.Marshal(responseData)
		w.Write(res)
		return
	}
	responseData.Successful = true
	responseData.Message = "Successfully Deleted Review"
	res, _ := json.Marshal(responseData)
	w.Write(res)
}

var GetReviews = func(w http.ResponseWriter, r *http.Request) {
	type ReviewResponse struct {
		Response
		Reviews []modules.UserReview `json:"reviews"`
	}

	userID, err := strconv.ParseInt(r.URL.Query().Get("discordid"), 10, 64)
	reviews, err := modules.GetReviews(userID)
	response := ReviewResponse{}

	if slices.Contains(common.OptedOut, uint64(userID)) {
		reviews := append([]database.UserReview{{
			ID:              0,
			SenderUsername:  "ReviewDB",
			ProfilePhoto:    "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
			Comment:         "This user has opted out of ReviewDB. It means you cannot review this user.",
			ReviewType:      3,
			SenderDiscordID: "287555395151593473",
			SystemMessage:   true,
			Badges:          []database.UserBadge{},
		}})
		jsonReviews, _ := json.Marshal(reviews)

		io.WriteString(w, string(jsonReviews))
		return
	}

	for i, j := 0, len(reviews)-1; i < j; i, j = i+1, j-1 {
		reviews[i], reviews[j] = reviews[j], reviews[i]
	}

	if err != nil {
		response.Successful = false
		response.Message = "An error occurred"
		common.SendStructResponse(w, response)
		return
	}

	if r.Header.Get("User-Agent") == "Aliucord (https://github.com/Aliucord/Aliucord)" && r.URL.Query().Get("noAds") != "true" {
		reviews = append([]modules.UserReview{{
			Comment:    "If you like the plugins I make, please consider supporting me at: \nhttps://github.com/sponsors/mantikafasi\n You can disable this in settings",
			ReviewType: 2,
			Sender: modules.Sender{
				DiscordID:    "287555395151593473",
				ProfilePhoto: "https://cdn.discordapp.com/attachments/527211215785820190/1079358371481800725/c4b7353e759983f5a3d686c7937cfab7.png?size=128",
				Username:     "ReviewDB",
			},
			SystemMessage: true,
		}}, reviews...)
	}

	if len(reviews) != 0 {
		reviews = append([]modules.UserReview{{
			ID:            0,
			Comment:       "Spamming and writing offensive reviews will result with a ban. Please be respectful to other users.",
			ReviewType:    3,
			SystemMessage: true,
			Sender: modules.Sender{
				DiscordID:    "287555395151593473",
				ProfilePhoto: "https://cdn.discordapp.com/attachments/1045394533384462377/1084900598035513447/646808599204593683.png?size=128",
				Username:     "Warning",
				Badges:       []database.UserBadge{},
			},
		}}, reviews...)
	}

	if reviews == nil { //we dont want to send null
		reviews = []modules.UserReview{}
	}

	response.Reviews = reviews
	response.Successful = true
	common.SendStructResponse(w, response)
}
