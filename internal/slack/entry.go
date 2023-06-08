package slack

import "time"

type SlackField struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short,omitempty"`
}

type SlackAttachment struct {
	Fallback   string       `json:"fallback,omitempty"`
	Color      string       `json:"color,omitempty"`
	PreText    string       `json:"pretext,omitempty"`
	AuthorName string       `json:"author_name,omitempty"`
	AuthorLink string       `json:"author_link,omitempty"`
	AuthorIcon string       `json:"author_icon,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	MarkdownIn []string     `json:"mrkdwn_in,omitempty"`
	ImageUrl   string       `json:"image_url,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon"`
	Fields     []SlackField `json:"fields,omitempty"`
}

type SlackMessage struct {
	Parse       string            `json:"parse,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Timestamp   time.Time         `json:"ts,omitempty"`
	Markdown    bool              `json:"mrkdwn,omitempty"`
	Channel     string            `json:"-"`
}
