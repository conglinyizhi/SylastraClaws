package discord

import (
	"github.com/conglinyizhi/SylastraClaws/pkg/audio/tts"
	"github.com/conglinyizhi/SylastraClaws/pkg/bus"
	"github.com/conglinyizhi/SylastraClaws/pkg/channels"
	"github.com/conglinyizhi/SylastraClaws/pkg/config"
)

func init() {
	channels.RegisterFactory(
		config.ChannelDiscord,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.DiscordSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			ch, err := NewDiscordChannel(bc, c, b)
			if err == nil {
				ch.tts = tts.DetectTTS(cfg)
			}
			return ch, err
		},
	)
}
