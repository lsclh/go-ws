package ws

/*
	//使用方式
	//先获取client实例
	c := ws.NewWebsocketClient(&ws.Config{
		WsHost:    &url.URL{Scheme: "ws", Host: "47.104.255.166:9502", Path: ""},
		KeepAlive: true,
		Reconnect: true,
		//Timeout:   time.Second * 20,
	})
	//按照需要设置好监听的闭包函数

	//ws连接成功的时候在这里显示
	c.OnOpen(func() {
		fmt.Println("ws 连接成功")
		//发送信息 支持 string []byte{}
		c.Send(`{"class":"Api\\Test","action":"who","content":{}}`)
	})
	//ws收到消息的时候在这里显示
	c.OnMessage(func(bytes []byte) {
		fmt.Println("ws 接受到消息: " + string(bytes))
	})
	//ws关闭的时候在这里显示
	c.OnClose(func() {
		fmt.Println("ws 已断开")
	})
	//连接过程中 读取 写入 等操作出现错误或异常在这里显示
	c.OnError(func(err error) {
		fmt.Println("ws 出现错误: " + err.Error())
	})
	//ws执行重连的时候在这里显示
	c.OnReconnect(func(err error) {
		fmt.Println("ws 重连中", err)
	})
	//执行连接操作
	if err := c.Connect(); err != nil {
		fmt.Println("ws 连接失败")
	}


*/

//创建一个websocket客户端
func NewWebsocketClient(conf *Config) (cl *client) {
	if conf.KeepAliveMessage == "" {
		conf.KeepAliveMessage = "ping"
	}
	cl = &client{
		connHost:         conf.WsHost,
		keepAlive:        conf.KeepAlive,
		isReconnect:      conf.Reconnect,
		timeout:          conf.Timeout,
		conn:             nil,
		isClose:          true,
		keepAliveMessage: []byte(conf.KeepAliveMessage),
		serviceType:      CLIENT,
	}
	return
}

//开始连接
func (c *client) Connect() error {
	if err := c.start(); err != nil {
		if c.isReconnect {
			go c.reconnect()
		}
		return err
	}
	return nil
}
