package messages

import "github.com/eclipse/paho.golang/autopaho"

type MqttMessage struct {
	Topic   string
	Payload string
}

type MqttStatus struct {
	Reason    string
	Code      byte
	Err       error
	Connected bool
}

type UrlParseError struct {
	URI string
	Err error
}

type MqttServerConnection struct {
	Connnection *autopaho.ConnectionManager
	Err         error
}

type Initialized byte

type PublishMessage struct {
	Payload string
	Topic   string
	Err     error
}

type SubscribeMessage struct {
	Topic string
	Err   error
}
