package service

import (
	"context"

	user "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/reva/v2/pkg/storagespace"
	"github.com/opencloud-eu/reva/v2/pkg/utils"

	ocEvents "github.com/opencloud-eu/opencloud/pkg/events"
	"github.com/opencloud-eu/opencloud/pkg/l10n"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/channels"
	"github.com/opencloud-eu/opencloud/services/notifications/pkg/email"
	"github.com/opencloud-eu/opencloud/services/settings/pkg/store/defaults"
)

func (s eventsNotifier) handleResourceMention(e ocEvents.ResourceMention, eventId string) {
	logger := s.logger.With().
		Str("event", "Mention").
		Str("resourceid", e.Ref.GetResourceId().GetOpaqueId()).
		Logger()
	_ = logger
	gatewayClient, err := s.gatewaySelector.Next()
	if err != nil {
		return
	}

	ctx, err := utils.GetServiceUserContextWithContext(context.Background(), gatewayClient, s.serviceAccountID, s.serviceAccountSecret)
	if err != nil {
		logger.Error().Err(err).Msg("could not select next gateway client")
		return
	}

	var data = struct {
		resourceLink string       `validate:"required,url"`
		resourceName string       `validate:"required,min=1"`
		author       *user.User   `validate:"required"`
		recipients   []*user.User `validate:"required,min=1"`
	}{}

	// fill the data struct with the info we need to render the email
	{
		resourceInfo, err := s.getResourceInfo(ctx, e.Ref.GetResourceId(), nil)
		if err != nil {
			return
		}
		data.resourceName = resourceInfo.GetName()

		data.resourceLink, err = urlJoinPath(s.openCloudURL, "f", storagespace.FormatResourceID(resourceInfo.GetId()))
		if err != nil {
			logger.Error().Err(err).Msg("failed to generate resource link.")
			return
		}

		for _, userID := range append([]*user.UserId{e.Executant}, e.UserIDs...) {
			switch u, err := s.getUser(ctx, userID); {
			case err != nil:
				logger.Error().Err(err).Msg("could not get user")
				return
			case userID.GetOpaqueId() == e.Executant.GetOpaqueId():
				data.author = u
			default:
				data.recipients = append(data.recipients, u)
			}
		}

		recipients := s.filter.execute(ctx, data.recipients, defaults.SettingUUIDProfileEventResourceMention)
		recipientsInstant, recipientsDaily, recipientsInstantWeekly := s.splitter.execute(ctx, recipients)
		recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalDaily, eventId, recipientsDaily)...)
		recipientsInstant = append(recipientsInstant, s.userEventStore.persist(_intervalWeekly, eventId, recipientsInstantWeekly)...)
		data.recipients = recipientsInstant
	}

	if err := validate.Struct(data); err != nil {
		logger.Error().Err(err).Msg("data struct validation failed")
		return
	}

	messages := make([]*channels.Message, len(data.recipients))
	for i, recipient := range data.recipients {
		locale := l10n.MustGetUserLocale(ctx, recipient.GetId().GetOpaqueId(), "", s.valueService)
		message, err := email.RenderEmailTemplate(email.Mention, locale, s.defaultLanguage, s.emailTemplatePath, s.translationPath, map[string]string{
			"AuthorName":    data.author.GetDisplayName(),
			"RecipientName": recipient.GetDisplayName(),
			"ResourceName":  data.resourceName,
			"ResourceLink":  data.resourceLink,
		})
		if err != nil {
			logger.Error().Err(err).Msg("could not render email-template")
			return
		}

		message.Sender = data.author.GetDisplayName()
		message.Recipient = []string{recipient.GetMail()}
		messages[i] = message
	}

	s.send(ctx, messages)
}
