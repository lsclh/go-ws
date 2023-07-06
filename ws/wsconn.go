package ws

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"sync"
)

// 配置文件
type Options struct {

	//作为客户端使用
	wsHost *url.URL

	//http写对象 作为服务端使用
	httpWrite http.ResponseWriter

	//http请求对象 作为服务端使用
	httpRequest *http.Request

	//连接成功回调
	onOpen onOpen

	//关闭回调
	onClose onClose

	//错误回调
	onError onError

	//接收到消息回调
	onMessage onMessage

	//连接类型
	serviceType service
}

type Option func(e *Options)

func WithServerHttp(w http.ResponseWriter, r *http.Request) Option {
	return func(e *Options) {
		e.httpWrite = w
		e.httpRequest = r
		e.serviceType = SERVER
	}
}

func WithClientWsUrl(u *url.URL) Option {
	return func(e *Options) {
		e.wsHost = u
		e.serviceType = CLIENT
	}
}

/*************************************************WS链接可操作函数start***************************************************/

// 获取ws对象
func NewWebsocket(opts ...Option) (cl *client) {
	cf := &Options{}
	for _, fn := range opts {
		fn(cf)
	}
	cl = &client{
		options: cf,
		isClose: true,
	}
	return
}

// 开始连接
func (c *client) Connect() error {
	if c.options.serviceType == CLIENT {
		conn, _, err := websocket.DefaultDialer.Dial(c.options.wsHost.String(), nil)
		if err != nil {
			return err
		}
		c.Conn = conn
	} else {
		conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.options.httpWrite, c.options.httpRequest, nil)
		if err != nil {
			http.NotFound(c.options.httpWrite, c.options.httpRequest)
			return err
		}
		c.Conn = conn
	}
	return c.init()
}

// 立即推送 -- 需要推送数量过多时 优先推送的数据
func (c *client) ForthwithSend(payload []byte) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = panicErr.(error)
		}
	}()
	c.sendChan <- payload
	return
}

// 稍后推送 -- 需要推送数量过多时 延后推送的数据
func (c *client) LaterSend(payload []byte) (err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = panicErr.(error)
		}
	}()
	c.secondSendChan <- payload
	return
}

// 断开连接
func (c *client) Close() {
	c.close()
}

/*************************************************WS链接可操作函数end***************************************************/

/*************************************************WS链接回调函数start***************************************************/

// 连接成功事件
func (c *client) OnOpen(fn onOpen) {
	c.options.onOpen = fn
}

// 断开连接
func (c *client) OnClose(fn onClose) {
	c.options.onClose = fn
}

// 连接期间发生错误
func (c *client) OnError(fn onError) {
	c.options.onError = fn
}

func (c *client) OnMessage(fn onMessage) {
	c.options.onMessage = fn
}

/*************************************************WS链接回调函数end***************************************************/

/*************************************************WS链接内部逻辑start***************************************************/

// 代码依赖项
type service int

const (
	CLIENT service = 1 << 0 //标记为客户端
	SERVER service = 1 << 1 //标记为服务端
)

type client struct {
	*websocket.Conn
	isClose        bool               //是否已关闭
	ctx            context.Context    //协程控制
	cancel         context.CancelFunc //协程控制
	sendChan       chan []byte        //主发送消息
	secondSendChan chan []byte        //次发送消息
	options        *Options

	lock sync.Mutex //锁

}

type onOpen func()

type onClose func()

type onMessage func([]byte)

type onError func(error)

var closeErrorCode = []int{
	websocket.CloseNormalClosure,
	websocket.CloseGoingAway,
	websocket.CloseProtocolError,
	websocket.CloseUnsupportedData,
	websocket.CloseNoStatusReceived,
	websocket.CloseAbnormalClosure,
	websocket.CloseInvalidFramePayloadData,
	websocket.ClosePolicyViolation,
	websocket.CloseMessageTooBig,
	websocket.CloseMandatoryExtension,
	websocket.CloseInternalServerErr,
	websocket.CloseServiceRestart,
	websocket.CloseTryAgainLater,
	websocket.CloseTLSHandshake,
}

// 实际执行开始的函数
func (c *client) init() (err error) {
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.sendChan = make(chan []byte, 100)
	c.secondSendChan = make(chan []byte, 100)
	c.isClose = false
	go c.writeWs()
	if c.options.onMessage != nil {
		go c.readWs()
	}
	c.openFn()
	return
}

// 实际执行结束的函数
func (c *client) close() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.isClose || c.Conn == nil {
		c.isClose = true
		return
	}
	c.isClose = true
	c.cancel()
	c.closeFn()

}

// 发送协程
func (c *client) writeWs() {
	defer func() {
		if err := recover(); err != nil {
			c.errorFn(fmt.Errorf("发送协程出现异常: %v", err))
		}
	}()
	for {
		select {
		case data := <-c.sendChan:
			c.write(data)
		case secondData := <-c.secondSendChan:
		priority:
			for {
				select {
				case data := <-c.sendChan:
					if len(data) == 0 {
						break priority
					}
					c.write(data)
				default:
					break priority
				}
			}
			c.write(secondData)
		case <-c.ctx.Done():
			c.isClose = true
		}
		if c.isClose {
			break
		}
	}
	_ = c.safeCloseChan(c.sendChan)
	for data := range c.sendChan {
		_ = c.WriteMessage(websocket.TextMessage, data)
	}
	_ = c.safeCloseChan(c.secondSendChan)
	for data := range c.secondSendChan {
		_ = c.WriteMessage(websocket.TextMessage, data)
	}
	_ = c.Conn.Close()
}

// 接收消息
func (c *client) readWs() {
	defer func() {
		if err := recover(); err != nil {
			c.errorFn(fmt.Errorf("读取协程出现异常: %v", err))
		}
	}()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			c.errorFn(err)
			if websocket.IsCloseError(err, closeErrorCode...) || c.isClose {
				c.close()
				return
			}
			continue
		}
		c.options.onMessage(message)
	}
}

func (c *client) write(data []byte) {
	err := c.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		c.errorFn(err)
		if websocket.IsCloseError(err, closeErrorCode...) {
			c.close()
			return
		}
	}
}

func (c *client) safeCloseChan(ch chan []byte) (justClosed bool) {
	defer func() {
		if recover() != nil {
			justClosed = false
		}
	}()
	// assume ch != nil here.
	close(ch)   // panic if ch is closed
	return true // <=> justClosed = true; return
}

// 执行结束闭包函数
func (c *client) errorFn(err error) {
	if c.options.onError != nil {
		c.options.onError(err)
	}
}

// 执行连接成功闭包函数
func (c *client) openFn() {
	if c.options.onOpen != nil {
		c.options.onOpen()
	}
}

// //执行关闭闭包函数
func (c *client) closeFn() {
	if c.options.onClose != nil {
		c.options.onClose()
	}
}

/*************************************************WS链接内部逻辑end***************************************************/
