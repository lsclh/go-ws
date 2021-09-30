package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
	"sync"
	"time"
)

//固定配置项
const (
	reconnectTime = time.Second * 10 //尝试重连时间
	keepAliveTime = time.Second * 10 //发送心跳时间
)

//配置文件
type Config struct {
	//连接ws的地址 &url.URL{Scheme: "ws", Host: "47.104.255.166:9502", Path: ""},
	//服务端使用不需要设置为nil
	WsHost *url.URL

	//读取超时时间 不设置表示不开启
	//建议无论客户端还是服务端都进行开启
	//定时发送心跳与回包 接收超时 说明链接出现异常 进行自动断开 如果是客户端进行重连防止异常死链
	Timeout time.Duration

	//断开是否重连
	//服务端使用不需要 设置为false 客户端建议开启
	Reconnect bool

	//是否使用自动心跳
	//服务端不需要使用心跳 设置为false 接收到消息回个包即可
	KeepAlive bool

	//心跳发送内容
	//服务端不需要使用心跳 设置为""
	KeepAliveMessage string
}

/*************************************************WS链接可操作函数start***************************************************/

//获取系统原始ws实例
func (c *client) GetClient() *websocket.Conn {
	return c.conn
}

//发送消息
func (c *client) Send(message interface{}) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = panicErr.(error)
		}
	}()
	var payload []byte
	switch message.(type) {
	case []byte:
		payload = message.([]byte)
	case string:
		payload = []byte(message.(string))
	default:
		return fmt.Errorf("错误的消息类型")
	}
	c.sendChan <- payload
	return
}

//断开连接
func (c *client) Close() {
	c.isReconnect = false
	c.close()
}

/*************************************************WS链接可操作函数end***************************************************/

/*************************************************WS链接回调函数start***************************************************/

//连接成功事件
func (c *client) OnOpen(fn onOpen) {
	if c.serviceType == SERVER {
		return
	}
	if c.onOpen == nil {
		c.onOpen = fn
	}
}

//触发重连事件
func (c *client) OnReconnect(fn onReconnect) {
	if c.serviceType == SERVER {
		return
	}
	if c.onReconnect == nil {
		c.onReconnect = fn
	}
}

//接收消息
func (c *client) OnMessage(fn onMessage) {
	c.onMessage = fn
}

//断开连接
func (c *client) OnClose(fn onClose) {
	c.onClose = fn
}

//连接期间发生错误
func (c *client) OnError(fn onError) {
	c.onError = fn
}

/*************************************************WS链接回调函数end***************************************************/

/*************************************************WS链接内部逻辑start***************************************************/

//代码依赖项
type service int

const (
	CLIENT service = 1 << 0 //标记为客户端
	SERVER service = 1 << 1 //标记为服务端
)

type client struct {
	connHost    *url.URL
	conn        *websocket.Conn
	isClose     bool          //是否已关闭
	closeChan   chan struct{} //主动关闭
	sendChan    chan []byte   //发送消息
	isReconnect bool          //是否需要重连
	timeout     time.Duration //读取超时时间

	onOpen      onOpen      //连接成功
	onMessage   onMessage   //消息接收
	onClose     onClose     //关闭回调
	onError     onError     //错误回调
	onReconnect onReconnect //错误回调

	lock sync.Mutex //锁

	keepAlive        bool    //是否发送心跳
	keepAliveMessage []byte  //发送心跳内容
	serviceType      service //连接类型
}

type onOpen func()

type onMessage func([]byte)

type onClose func()

type onError func(error)

type onReconnect func(error)

//实际执行开始的函数
func (c *client) start() (err error) {
	conn, _, err := websocket.DefaultDialer.Dial(c.connHost.String(), nil)
	if err != nil {
		return
	}
	c.conn = conn
	c.ws()
	c.openFn()
	return
}

func (c *client) ws() {
	c.closeChan = make(chan struct{})
	c.sendChan = make(chan []byte, 100)
	c.isClose = false
	go c.writeWs()
	go c.readWs()
}

//实际执行结束的函数
func (c *client) close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.isClose {
		return
	}
	c.isClose = true
	close(c.closeChan) //关闭管道通知其他协程退出
	close(c.sendChan)  //关闭发送数据通道
	if err := c.conn.Close(); err != nil {
		c.errorFn(fmt.Errorf("关闭链接错误: %s", err.Error()))
		return
	}
	c.conn = nil
	c.closeFn()
	if c.isReconnect {
		go c.reconnect()
	}
}

//实际执行重联的函数
func (c *client) reconnect() {
	defer func() {
		if err := recover(); err != nil {
			c.errorFn(fmt.Errorf("重连协程出现异常: %v", err))
		}
	}()
	for {
		if !c.isReconnect {
			return
		}
		err := c.start()
		c.reconnectFn(err)
		if err == nil {
			return
		}
		time.Sleep(reconnectTime)
	}
}

//读取协程
func (c *client) readWs() {
	defer func() {
		if err := recover(); err != nil {
			c.errorFn(fmt.Errorf("读取协程出现异常: %v", err))
		}
	}()
	for {
		if c.timeout.Seconds() > 0 {
			_ = c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		}
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.errorFn(err)
			c.close()
			return
		}
		c.messageFn(message)
	}
}

//发送协程
func (c *client) writeWs() {
	defer func() {
		if err := recover(); err != nil {
			c.errorFn(fmt.Errorf("发送协程出现异常: %v", err))
		}
	}()
	if !c.keepAlive {
		for {
			select {
			case msg := <-c.sendChan:
				err := c.conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					c.errorFn(err)
					c.closeFn()
					return
				}
			case <-c.closeChan:
				return
			}
		}
	} else {
		t := time.NewTicker(keepAliveTime)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				err := c.conn.WriteMessage(websocket.TextMessage, c.keepAliveMessage)
				if err != nil {
					c.errorFn(err)
					c.close()
					return
				}
			case msg := <-c.sendChan:
				err := c.conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					c.errorFn(err)
					c.close()
					return
				}
			case <-c.closeChan:
				return
			}
		}
	}

}

//执行结束闭包函数
func (c *client) errorFn(err error) {
	if c.onError != nil {
		c.onError(err)
	}
}

//执行连接成功闭包函数
func (c *client) openFn() {
	if c.onOpen != nil {
		c.onOpen()
	}
}

////执行关闭闭包函数
func (c *client) closeFn() {
	if c.onClose != nil {
		c.onClose()
	}
}

//执行发送消息闭包函数
func (c *client) messageFn(msg []byte) {
	if c.onMessage != nil {
		c.onMessage(msg)
	}
}

//执行重连闭包函数
func (c *client) reconnectFn(err error) {
	if c.onReconnect != nil {
		c.onReconnect(err)
	}
}

/*************************************************WS链接内部逻辑end***************************************************/
