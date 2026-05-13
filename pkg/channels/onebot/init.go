package onebot

import (
	"github.com/conglinyizhi/SylastraClaws/pkg/bus"
	"github.com/conglinyizhi/SylastraClaws/pkg/channels"
	"github.com/conglinyizhi/SylastraClaws/pkg/config"
)

func init() {
	channels.RegisterFactory(
		config.ChannelOneBot,
		func(channelName, channelType string, cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
			bc := cfg.Channels[channelName]
			decoded, err := bc.GetDecoded()
			if err != nil {
				return nil, err
			}
			c, ok := decoded.(*config.OneBotSettings)
			if !ok {
				return nil, channels.ErrSendFailed
			}
			return NewOneBotChannel(bc, c, b)
		},
	)
}
