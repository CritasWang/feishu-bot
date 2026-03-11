package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// Replier 飞书消息回复器
type Replier struct {
	client *lark.Client
}

func NewReplier(client *lark.Client) *Replier {
	return &Replier{client: client}
}

// Reply 回复一条消息，返回新消息的 messageID
func (r *Replier) Reply(messageID, text string) (string, error) {
	content, _ := json.Marshal(map[string]string{"text": text})

	resp, err := r.client.Im.Message.Reply(context.Background(),
		larkim.NewReplyMessageReqBuilder().
			MessageId(messageID).
			Body(larkim.NewReplyMessageReqBodyBuilder().
				MsgType(larkim.MsgTypeText).
				Content(string(content)).
				Build()).
			Build())

	if err != nil {
		log.Printf("回复消息失败: %v", err)
		return "", err
	}

	if !resp.Success() {
		log.Printf("回复消息失败: code=%d msg=%s", resp.Code, resp.Msg)
		return "", fmt.Errorf("feishu API error: %d %s", resp.Code, resp.Msg)
	}

	msgID := ""
	if resp.Data != nil && resp.Data.MessageId != nil {
		msgID = *resp.Data.MessageId
	}
	return msgID, nil
}

// Update 更新已发送的消息内容
func (r *Replier) Update(messageID, text string) error {
	content, _ := json.Marshal(map[string]string{"text": text})

	resp, err := r.client.Im.Message.Patch(context.Background(),
		larkim.NewPatchMessageReqBuilder().
			MessageId(messageID).
			Body(larkim.NewPatchMessageReqBodyBuilder().
				Content(string(content)).
				Build()).
			Build())

	if err != nil {
		return err
	}

	if !resp.Success() {
		return fmt.Errorf("feishu API error: %d %s", resp.Code, resp.Msg)
	}

	return nil
}

// SendToChat 主动向聊天发送消息
func (r *Replier) SendToChat(chatID, text string) (string, error) {
	content, _ := json.Marshal(map[string]string{"text": text})

	resp, err := r.client.Im.Message.Create(context.Background(),
		larkim.NewCreateMessageReqBuilder().
			ReceiveIdType(larkim.ReceiveIdTypeChatId).
			Body(larkim.NewCreateMessageReqBodyBuilder().
				MsgType(larkim.MsgTypeText).
				ReceiveId(chatID).
				Content(string(content)).
				Build()).
			Build())

	if err != nil {
		return "", err
	}

	if !resp.Success() {
		return "", fmt.Errorf("feishu API error: %d %s", resp.Code, resp.Msg)
	}

	msgID := ""
	if resp.Data != nil && resp.Data.MessageId != nil {
		msgID = *resp.Data.MessageId
	}
	return msgID, nil
}
