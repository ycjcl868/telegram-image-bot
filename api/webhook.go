package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

var bot *tgbotapi.BotAPI
var imgHost string
var githubRepo string
var githubToken string

func init() {
	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	githubToken = os.Getenv("GITHUB_TOKEN")
	imgHost = os.Getenv("IMG_HOST")
	githubRepo = os.Getenv("GITHUB_REPO")
	newBot, err := tgbotapi.NewBotAPI(tgToken)

	if err != nil {
		log.Panic(err)
	}

	bot = newBot
	log.Printf("Authorized on account %s", bot.Self.UserName)
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

	updateStr, _ := json.Marshal(update)
	fmt.Printf("%s\n\n", updateStr)

	data := Response{
		Msg:       "Hello",
		Method:    "sendMessage",
		ChatID:    update.Message.Chat.ID,
		ReplyToID: update.Message.MessageID,
	}

	if update.Message.IsCommand() {
		return
	}

	if len(update.Message.Photo) > 0 {
		imgUrl, err := doUpload(&update)
		if err != nil {
			data.Msg = err.Error()
		} else {
			data.Msg = imgUrl
		}
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
	fmt.Fprint(w, string(msg))
}

func doUpload(update *tgbotapi.Update) (string, error) {
	fmt.Printf("photo: %d\n", len(update.Message.Photo))
	fileId := update.Message.Photo[len(update.Message.Photo)-1].FileID
	fmt.Printf("fileId: %s\n", fileId)
	imgUrl, err := bot.GetFileDirectURL(fileId)
	fmt.Printf("imgUrl: %s \n", imgUrl)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	resp, err := http.Get(imgUrl)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	imgBase64 := base64.StdEncoding.EncodeToString(bytes)

	filename := fmt.Sprintf("%s%s", gonanoid.Must(), path.Ext(imgUrl))
	date := time.Unix(int64(update.Message.Date), 0)
	prefix := fmt.Sprintf("%02d/%02d", date.Year(), date.Month())
	filePath := path.Join(prefix, filename)
	fmt.Printf("filePath: %s\n", filePath)

	githubRes, err := uploadToGithub(filePath, imgBase64)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	if githubRes.Content.Path != "" {
		filename = githubRes.Content.Path
	}

	imgPaths := []string{
		fmt.Sprintf("%s/%s", imgHost, filename),
	}

	if githubRes.Content.DownloadURL != "" {
		imgPaths = append(imgPaths, githubRes.Content.DownloadURL)
	}

	imgPathStr := strings.Join(imgPaths, "\n\n")

	fmt.Println(imgPaths)
	fmt.Println(imgPathStr)
	return imgPathStr, nil
}

func uploadToGithub(filePath string, content string) (GithubResponse, error) {
	filenameEncoding := url.QueryEscape(filePath)
	githubUrl := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", githubRepo, filenameEncoding)
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	payload := strings.NewReader(fmt.Sprintf(`{
   "message": "Upload by Telegram",
    "branch": "main",
    "content": "%s",
    "path": "%s"
}`, content, filenameEncoding))

	respResp := GithubResponse{}

	client := &http.Client{}
	req, err := http.NewRequest("PUT", githubUrl, payload)

	if err != nil {
		fmt.Println(err)
		return respResp, err
	}
	req.Header.Add("User-Agent", "Telegram")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", githubToken))
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
	if err != nil {
		fmt.Println(err)
		return respResp, err
	}

	return respResp, nil
}
