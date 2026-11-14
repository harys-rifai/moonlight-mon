package hub

import (
	"net/mail"

	"github.com/henrygd/beszel/internal/alerts"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// NotifyDBAlert delivers a database observability alert to admin notification
// channels (webhooks and email) defined in user settings.
func (h *Hub) NotifyDBAlert(title, message string) {
	settingsRecords, err := h.FindAllRecords("user_settings")
	if err != nil {
		return
	}
	for _, settingsRecord := range settingsRecords {
		user, err := h.FindRecordById("users", settingsRecord.GetString("user"))
		if err != nil || user.GetString("role") != "admin" {
			continue
		}
		var settings alerts.UserNotificationSettings
		if err := settingsRecord.UnmarshalJSONField("settings", &settings); err != nil {
			continue
		}
		for _, webhook := range settings.Webhooks {
			if webhook == "" {
				continue
			}
			if err := h.SendShoutrrrAlert(webhook, title, message, "", ""); err != nil {
				h.Logger().Error("dbobs: failed to send webhook alert", "err", err)
			}
		}
		if len(settings.Emails) == 0 {
			continue
		}
		to := make([]mail.Address, 0, len(settings.Emails))
		for _, address := range settings.Emails {
			if address != "" {
				to = append(to, mail.Address{Address: address})
			}
		}
		if len(to) == 0 {
			continue
		}
		meta := h.Settings().Meta
		msg := &mailer.Message{
			From:    mail.Address{Address: meta.SenderAddress, Name: meta.SenderName},
			To:      to,
			Subject: title,
			Text:    message,
		}
		if err := h.NewMailClient().Send(msg); err != nil {
			h.Logger().Error("dbobs: failed to send email alert", "err", err)
		}
	}
}

