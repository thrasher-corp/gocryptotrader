package telegram

// User holds user information
type User struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
	Result      struct {
		ID           int64  `json:"id"`
		IsBot        bool   `json:"is_bot"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		UserName     string `json:"username"`
		LanguageCode string `json:"language_code"`
	} `json:"result"`
}

// GetUpdateResponse represents an incoming update
type GetUpdateResponse struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
	Result      []struct {
		UpdateID           int64       `json:"update_id"`
		Message            MessageType `json:"message"`
		EditedMessage      any         `json:"edited_message"`
		ChannelPost        any         `json:"channel_post"`
		EditedChannelPost  any         `json:"edited_channel_post"`
		InlineQuery        any         `json:"inline_query"`
		ChosenInlineResult any         `json:"chosen_inline_result"`
		CallbackQuery      any         `json:"callback_query"`
		ShippingQuery      any         `json:"shipping_query"`
		PreCheckoutQuery   any         `json:"pre_checkout_query"`
	} `json:"result"`
}

// Message holds the full message information
type Message struct {
	Ok          bool        `json:"ok"`
	Description string      `json:"description"`
	Result      MessageType `json:"result"`
}

// MessageType contains message data
type MessageType struct {
	MessageID            int64               `json:"message_id"`
	From                 UserType            `json:"from"`
	Date                 int64               `json:"date"`
	Chat                 ChatType            `json:"chat"`
	ForwardFrom          UserType            `json:"forward_from"`
	ForwardFromChat      ChatType            `json:"forward_from_chat"`
	ForwardFromMessageID int64               `json:"forward_from_message_id"`
	ForwardSignature     string              `json:"forward_signature"`
	ForwardDate          int64               `json:"forward_date"`
	ReplyToMessage       any                 `json:"reply_to_message"`
	EditDate             int64               `json:"edit_date"`
	MediaGroupID         string              `json:"media_group_id"`
	AuthorSignature      string              `json:"author_signature"`
	Text                 string              `json:"text"`
	Entities             []MessageEntityType `json:"entities"`
	CaptionEntities      []MessageEntityType `json:"caption_entities"`
}

// MessageEntityType contains message entity information
type MessageEntityType struct {
	Type   string   `json:"type"`
	Offset int64    `json:"offset"`
	Length int64    `json:"length"`
	URL    string   `json:"url"`
	User   UserType `json:"user"`
}

// UserType contains user data
type UserType struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	UserName     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// ChatType contains chat data
type ChatType struct {
	ID               int64  `json:"id"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	UserName         string `json:"username"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	AllAdmin         bool   `json:"all_members_are_administrators"`
	Description      string `json:"description"`
	InviteLink       string `json:"invite_link"`
	StickerSetName   string `json:"sticker_set_name"`
	CanSetStickerSet bool   `json:"can_set_sticker_set"`
}
