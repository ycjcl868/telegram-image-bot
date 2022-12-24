package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Response struct {
	Msg       string `json:"text"`
	ChatID    int64  `json:"chat_id"`
	Method    string `json:"method"`
	ReplyToID int    `json:"reply_to_message_id"`
}

type GithubResponse struct {
	Content struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Sha         string `json:"sha"`
		Size        int    `json:"size"`
		URL         string `json:"url"`
		HTMLURL     string `json:"html_url"`
		GitURL      string `json:"git_url"`
		DownloadURL string `json:"download_url"`
		Type        string `json:"type"`
		Links       struct {
			Self string `json:"self"`
			Git  string `json:"git"`
			HTML string `json:"html"`
		} `json:"_links"`
	} `json:"content"`
	Commit struct {
		Sha     string `json:"sha"`
		NodeID  string `json:"node_id"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		Author  struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
		Tree struct {
			Sha string `json:"sha"`
			URL string `json:"url"`
		} `json:"tree"`
		Message string `json:"message"`
		Parents []struct {
			Sha     string `json:"sha"`
			URL     string `json:"url"`
			HTMLURL string `json:"html_url"`
		} `json:"parents"`
		Verification struct {
			Verified  bool        `json:"verified"`
			Reason    string      `json:"reason"`
			Signature interface{} `json:"signature"`
			Payload   interface{} `json:"payload"`
		} `json:"verification"`
	} `json:"commit"`
}

var bot *tgbotapi.BotAPI

func init() {
	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	newBot, err := tgbotapi.NewBotAPI(tgToken)

	if err != nil {
		log.Panic(err)
	}

	bot = newBot
}

func Handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// 读取请求 body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%s\n\n", string(body))

	var update tgbotapi.Update // 创建一个 bot update

	err = json.Unmarshal(body, &update)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	fmt.Printf("555555555555555555\n")

	updateStr, _ := json.Marshal(update)
	fmt.Printf("%s\n\n", updateStr)

	fmt.Printf("66666666666666666666\n")

	data := Response{
		Msg:       "Hello",
		Method:    "sendMessage",
		ChatID:    update.Message.Chat.ID,
		ReplyToID: update.Message.MessageID,
	}

	fmt.Printf("photo: %s\n", len(update.Message.Photo))

	if len(update.Message.Photo) > 0 {
		fileId := update.Message.Photo[len(update.Message.Photo)-1].FileID
		fmt.Printf("fileId: %s\n", fileId)
		imgUrl, err := bot.GetFileDirectURL(fileId)
		fmt.Printf("imgUrl: %s \n", imgUrl)
		if err != nil {
			log.Fatalln(err)
		}

		resp, err := http.Get(imgUrl)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		imgBase64 := base64.StdEncoding.EncodeToString(bytes)
		filename := fmt.Sprintf("%s%s", strconv.Itoa(update.Message.Date), path.Ext(imgUrl))
		fmt.Printf("filename: %s\n", filename)
		githubRes, err := uploadToGithub(filename, imgBase64)
		if err != nil {
			log.Fatalln(err)
		}

		imgPath := filename
		if githubRes.Content.Path != "" {
			imgPath = githubRes.Content.Path
		}

		imgPaths := []string{
			fmt.Sprintf("https://images.rustc.cloud/%s", imgPath),
		}

		if githubRes.Content.DownloadURL != "" {
			imgPaths = append(imgPaths, githubRes.Content.DownloadURL)
		}

		imgPathStr := strings.Join(imgPaths, "\n\n")

		fmt.Println(imgPaths)
		fmt.Println(imgPathStr)

		data.Msg = imgPathStr
	} else if update.Message.Text != "" {
		data.Msg = update.Message.Text
		// 在控制台打印收到的消息
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	}

	msg, _ := json.Marshal(data)
	// 在控制台打印响应内容
	log.Printf("Response %s", string(msg))
	// 向响应头添加 Content-Type
	w.Header().Add("Content-Type", "application/json")
	// 发送格式化输出
	fmt.Fprintf(w, string(msg))
}

func uploadToGithub(filename string, content string) (GithubResponse, error) {
	filenameEncoding := url.QueryEscape(filename)
	githubUrl := fmt.Sprintf("https://api.github.com/repos/ycjcl868/images/contents/%s", filenameEncoding)
	method := "PUT"
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	payload := strings.NewReader(fmt.Sprintf(`{
   "message": "Upload by Telegram",
    "branch": "main",
    "content": "%s",
    "path": "%s"
}`, content, filenameEncoding))

	respResp := GithubResponse{}

	client := &http.Client{}
	req, err := http.NewRequest(method, githubUrl, payload)

	if err != nil {
		fmt.Println(err)
		return respResp, err
	}
	req.Header.Add("User-Agent", "Telegram")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", os.Getenv("GITHUB_TOKEN")))
	req.Header.Add("Content-Type", mimeType)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return respResp, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return respResp, err
	}
	fmt.Println(string(body))

	err = json.Unmarshal(body, &respResp)

	return respResp, nil
}
