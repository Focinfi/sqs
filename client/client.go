package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Focinfi/sqs/errors"
	"github.com/Focinfi/sqs/log"
	"github.com/Focinfi/sqs/models"
	"github.com/Focinfi/sqs/util/urlutil"
)

const (
	jsonHTTPHeader = "application/json"
	authCodeKey    = "auth_code"

	applyNodeURLFormat               = "%s/applyNode"
	applyMessageIDURLFormat          = "%s/messageID"
	pushMessageURLFormat             = "%s/message"
	pullMessageURLFormat             = "%s/messages"
	reportReceivedMessageIDURLFormat = "%s/receivedMessageID"

	// DefaultSquad is the default squad name
	DefaultSquad = "default"
)

// Option for Client options
type Option struct {
	// Endpoint for main server
	Endpoint string
	models.UserAuth
}

// Client for one sqs client
type Client struct {
	opt *Option
}

// New allocates a new Client
func New(endpoint string, accessKey string, secretKey string) *Client {
	return &Client{
		opt: &Option{
			Endpoint: endpoint,
			UserAuth: models.UserAuth{
				AccessKey: accessKey,
				SecretKey: secretKey,
			},
		},
	}
}

// QueueClient for one query client
type QueueClient struct {
	*Client
	servingNode string
	BaseInfo
}

// BaseInfo for one client basic info
type BaseInfo struct {
	Token     string `json:"token"`
	QueueName string `json:"queue_name"`
	SquadName string `json:"squad_name,omitempty"`
}

type registerResponseParam struct {
	Token string `json:"token,omitempty"`
	Node  string `json:"node"`
}

type pushMessageParam struct {
	BaseInfo
	MessageID int64  `json:"message_id"`
	Content   string `json:"content"`
}

type applyMessageIDParam struct {
	BaseInfo
	Size int `json:"size"`
}

type applyMessageResponseParam struct {
	MessageIDBegin int64 `json:"message_id_begin"`
	MessageIDEnd   int64 `json:"message_id_end"`
}

type reportReceivedParam struct {
	BaseInfo
	MessageID int64 `json:"message_id"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Content   string `json:"content"`
}

// Queue returns a new QueueClient with the given name
func (cli *Client) Queue(name string, squad string) (*QueueClient, error) {
	if name == "" {
		return nil, errors.New("queue can not be empty")
	}

	if squad == "" {
		squad = DefaultSquad
	}

	return &QueueClient{
		Client: cli,
		BaseInfo: BaseInfo{
			QueueName: name,
			SquadName: squad,
		},
	}, nil
}

// ApplyNode applies for a node
func (cli *QueueClient) ApplyNode() error {
	aplyParams := &struct {
		models.UserAuth
		BaseInfo
	}{
		UserAuth: cli.Client.opt.UserAuth,
		BaseInfo: cli.BaseInfo,
	}

	b, err := json.Marshal(aplyParams)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(applyNodeURLFormat, urlutil.MakeURL(cli.Client.opt.Endpoint))
	resp, err := http.Post(url, jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	type param struct {
		models.HTTPStatusMeta
		Data registerResponseParam
	}
	respData := &param{}
	if err := json.Unmarshal(respBytes, respData); err != nil {
		return err
	}

	log.Biz.Infoln(string(respBytes))
	data := respData.Data
	if data.Node == "" || data.Token == "" {
		return errors.New("failed to register for a server IP")
	}

	cli.servingNode = data.Node
	cli.BaseInfo.Token = data.Token
	return nil
}

// PushMessage pushes a message
func (cli *QueueClient) PushMessage(content string) error {
	// apply a id
	id, err := cli.applyMessageID()
	if err != nil {
		return err
	}

	log.Internal.Infoln("applyMessageID:", id)

	param := &pushMessageParam{
		MessageID: id,
		Content:   content,
		BaseInfo:  cli.BaseInfo,
	}

	b, err := json.Marshal(param)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(pushMessageURLFormat, urlutil.MakeURL(cli.servingNode))
	resp, err := http.Post(url, jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to push message, resonse status code: %d\n", resp.StatusCode)
	}

	return nil
}

// PullMessage for pull message request
func (cli *QueueClient) PullMessage(handler func([]Message) error) error {
	reqBytes, err := json.Marshal(cli.BaseInfo)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(pullMessageURLFormat, urlutil.MakeURL(cli.servingNode))
	resp, err := http.Post(url, jsonHTTPHeader, bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("filed to pull message, status code is %d\n", resp.StatusCode)
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	respParam := &struct {
		models.HTTPStatusMeta
		Data struct {
			Messages []Message `json:"messages"`
		}
	}{}
	if err := json.Unmarshal(respBytes, respParam); err != nil {
		return err
	}
	messages := respParam.Data.Messages
	if len(messages) > 0 {
		log.Internal.Infoln(messages)
		if err := handler(messages); err != nil {
			return err
		}

		go cli.reportReceived(messages[len(messages)-1].MessageID)
	}

	return nil
}

// reportReceived reports the last received message id
func (cli *QueueClient) reportReceived(messageID int64) error {
	url := fmt.Sprintf(reportReceivedMessageIDURLFormat, urlutil.MakeURL(cli.servingNode))
	param := &reportReceivedParam{
		BaseInfo:  cli.BaseInfo,
		MessageID: messageID,
	}

	b, err := json.Marshal(param)
	if err != nil {
		return err
	}

	var delay time.Duration

	for {
		select {
		case <-time.After(delay):
			resp, err := http.Post(url, jsonHTTPHeader, bytes.NewReader(b))
			if err != nil {
				log.Service.Errorf("can not report received message id, err: %v\n", err)
				delay = (delay + 1) * time.Millisecond * 500
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Service.Errorf("can not report received message id, status code: %d\n", resp.StatusCode)
				continue
			}

			return nil
		}
	}
}

func (cli *QueueClient) applyMessageID() (int64, error) {
	param := &applyMessageIDParam{cli.BaseInfo, 1}
	b, err := json.Marshal(param)
	if err != nil {
		return -1, err
	}

	url := fmt.Sprintf(applyMessageIDURLFormat, urlutil.MakeURL(cli.servingNode))
	resp, err := http.Post(url, jsonHTTPHeader, bytes.NewReader(b))
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	respData := &struct {
		BaseInfo
		Data applyMessageResponseParam
	}{}
	if err := json.Unmarshal(respBytes, respData); err != nil {
		return -1, err
	}

	if respData.Data.MessageIDEnd < respData.Data.MessageIDBegin {
		return -1, errors.New("GET /appyMessageID response data broken: end < begin")
	}

	return respData.Data.MessageIDEnd, nil
}
