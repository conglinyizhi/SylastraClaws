// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package channels

import (
	"context"

	"github.com/conglinyizhi/SylastraClaws/pkg/bus"
	"github.com/conglinyizhi/SylastraClaws/pkg/config"
	"github.com/conglinyizhi/SylastraClaws/pkg/logger"
	"github.com/conglinyizhi/SylastraClaws/pkg/media"
)

func NewManager(cfg *config.Config, messageBus *bus.MessageBus, store media.MediaStore) (*Manager, error) {
	m := &Manager{
		channels:      make(map[string]Channel),
		workers:       make(map[string]*channelWorker),
		bus:           messageBus,
		config:        cfg,
		mediaStore:    store,
		channelHashes: make(map[string]string),
	}

	// Register as streaming delegate so the agent loop can obtain streamers
	messageBus.SetStreamDelegate(m)

	if err := m.initChannels(&cfg.Channels); err != nil {
		return nil, err
	}

	// Store initial config hashes for all channels
	m.channelHashes = toChannelHashes(cfg)

	return m, nil
}

// GetStreamer implements bus.StreamDelegate.
// It checks if the named channel supports streaming and returns a Streamer.
func (m *Manager) GetStreamer(ctx context.Context, channelName, chatID string) (bus.Streamer, bool) {
	m.mu.RLock()
	ch, exists := m.channels[channelName]
	m.mu.RUnlock()

	if !exists {
		return nil, false
	}

	sc, ok := ch.(StreamingCapable)
	if !ok {
		return nil, false
	}

	streamer, err := sc.BeginStream(ctx, chatID)
	if err != nil {
		logger.DebugCF("channels", "Streaming unavailable, falling back to placeholder", map[string]any{
			"channel": channelName,
			"error":   err.Error(),
		})
		return nil, false
	}

	// Mark streamActive on Finalize so preSend knows to clean up the placeholder
	key := channelName + ":" + chatID
	return &finalizeHookStreamer{
		Streamer: streamer,
		onFinalize: func(finalizeCtx context.Context) {
			if m.toolFeedbackSeparateMessagesEnabled() {
				clearTrackedToolFeedbackMessage(
					ch,
					chatID,
					&bus.InboundContext{
						Channel: channelName,
						ChatID:  chatID,
					},
				)
			} else {
				dismissTrackedToolFeedbackMessage(
					finalizeCtx,
					ch,
					chatID,
					&bus.InboundContext{
						Channel: channelName,
						ChatID:  chatID,
					},
				)
			}
			m.streamActive.Store(key, true)
		},
	}, true
}

// finalizeHookStreamer wraps a Streamer to run a hook on Finalize.
type finalizeHookStreamer struct {
	Streamer
	onFinalize func(context.Context)
}

func (s *finalizeHookStreamer) Finalize(ctx context.Context, content string) error {
	if err := s.Streamer.Finalize(ctx, content); err != nil {
		return err
	}
	if s.onFinalize != nil {
		s.onFinalize(ctx)
	}
	return nil
}

// initChannel is a helper that looks up a factory by type name and creates the channel.
// typeName is the channel type used for factory lookup (e.g., "telegram").
// channelName is the config map key used as the channel's runtime name (e.g., "my_telegram").
func (m *Manager) initChannel(typeName, channelName string) {
	f, ok := getFactory(typeName)
	if !ok {
		logger.WarnCF("channels", "Factory not registered", map[string]any{
			"channel": channelName,
			"type":    typeName,
		})
		return
	}
	logger.DebugCF("channels", "Attempting to initialize channel", map[string]any{
		"channel": channelName,
		"type":    typeName,
	})
	ch, err := f(channelName, typeName, m.config, m.bus)
	if err != nil {
		logger.ErrorCF("channels", "Failed to initialize channel", map[string]any{
			"channel": channelName,
			"type":    typeName,
			"error":   err.Error(),
		})
	} else {
		// Inject MediaStore if channel supports it
		if m.mediaStore != nil {
			if setter, ok := ch.(interface{ SetMediaStore(s media.MediaStore) }); ok {
				setter.SetMediaStore(m.mediaStore)
			}
		}
		// Inject PlaceholderRecorder if channel supports it
		if setter, ok := ch.(interface{ SetPlaceholderRecorder(r PlaceholderRecorder) }); ok {
			setter.SetPlaceholderRecorder(m)
		}
		// Inject owner reference so BaseChannel.HandleMessage can auto-trigger typing/reaction
		if setter, ok := ch.(interface{ SetOwner(ch Channel) }); ok {
			setter.SetOwner(ch)
		}
		m.channels[channelName] = ch
		logger.InfoCF("channels", "Channel enabled successfully", map[string]any{
			"channel": channelName,
			"type":    typeName,
		})
	}
}

func (m *Manager) getChannelConfigAndEnabled(channelName string) (*config.Channel, bool) {
	bc, ok := m.config.Channels[channelName]
	if !ok || bc == nil {
		return nil, false
	}
	if !bc.Enabled {
		return bc, false
	}

	// Use Type to determine the config struct for validation.
	// The map key (channelName) is the config key, which may differ from the type.
	channelType := bc.Type
	if channelType == "" {
		channelType = channelName
	}

	// Settings have already been decoded by InitChannelList, so we just need to
	// type-assert and check the relevant fields.
	decoded, err := bc.GetDecoded()
	if err != nil {
		return bc, false
	}
	//nolint:revive
	switch settings := decoded.(type) {
	case *config.WhatsAppSettings:
		if channelType == config.ChannelWhatsApp {
			return bc, settings.BridgeURL != ""
		}
		return bc, channelType == config.ChannelWhatsAppNative && settings.UseNative
	case *config.MatrixSettings:
		return bc, settings.Homeserver != "" && settings.UserID != "" && settings.AccessToken.String() != ""
	case *config.WeComSettings:
		return bc, settings.BotID != "" && settings.Secret.String() != ""
	case *config.PicoClientSettings:
		return bc, settings.URL != ""
	case *config.DingTalkSettings:
		return bc, settings.ClientID != ""
	case *config.SlackSettings:
		return bc, settings.BotToken.String() != ""
	case *config.WeixinSettings:
		return bc, settings.Token.String() != ""
	case *config.PicoSettings:
		return bc, settings.Token.String() != ""
	case *config.IRCSettings:
		return bc, settings.Server != ""
	case *config.LINESettings:
		return bc, settings.ChannelAccessToken.String() != ""
	case *config.OneBotSettings:
		return bc, settings.WSUrl != ""
	case *config.QQSettings:
		return bc, settings.AppSecret.String() != ""
	case *config.TelegramSettings:
		return bc, settings.Token.String() != ""
	case *config.FeishuSettings:
		return bc, settings.AppSecret.String() != ""
	case *config.MaixCamSettings:
		return bc, true
	case *config.TeamsWebhookSettings:
		return bc, true
	case *config.DiscordSettings:
		return bc, settings.Token.String() != ""
	case *config.VKSettings:
		return bc, settings.GroupID != 0 && settings.Token.String() != ""
	}

	return bc, bc.Enabled
}

// initChannels initializes all enabled channels based on the configuration.
// It iterates config entries and uses bc.Type to look up the appropriate factory.
func (m *Manager) initChannels(channels *config.ChannelsConfig) error {
	logger.InfoC("channels", "Initializing channel manager")

	for name, bc := range *channels {
		if !bc.Enabled {
			continue
		}
		_, ready := m.getChannelConfigAndEnabled(name)
		if !ready {
			continue
		}
		typeName := bc.Type
		if typeName == "" {
			typeName = name
		}
		m.initChannel(typeName, name)
	}

	logger.InfoCF("channels", "Channel initialization completed", map[string]any{
		"enabled_channels": len(m.channels),
	})

	return nil
}

// SetupHTTPServer creates a shared HTTP server with the given listen address.
// It registers health endpoints from the health server and discovers channels
