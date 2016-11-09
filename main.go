package main

import (
    "fmt"
    "log"
    "os"
    // "reflect"
    // "sync"
    // "regexp"
    "strconv"
    // "errors"
    "strings"
    _ "github.com/lib/pq"
    "database/sql"

    "github.com/nlopes/slack"
)

type App struct{
    Api *slack.Client
    RTM *slack.RTM
    DB *sql.DB
    Rooms map[string]*Room
}

type Room struct{
    Id string
    Status Status
    Users []User
}

type Status struct{
    Phase int // 0:init, 1:朝, 2:村人投票, 3:人狼のターンと(占い師, 騎士)の選択, 4:処理,結果表示(1に行くか終わる)
    Days int // 日数
}

type User struct{
    Name string
    Channel string
    Job int // 0:None 1:村人, 2:人狼, 3:占い師, 4:騎士
    IsAlive bool
}

type RoomError struct{
    Msg string
    Code int
}

func (e *RoomError) Error() string {
    return "RoomError"
}

func (app *App) NewRoom(roomId string) error {
    newStatus := Status{
        Phase: 0,
        Days: 0,
    }
    newRoom := Room{
        Id: roomId,
        Status: newStatus,
        Users: make([]User, 0),
    }
    app.Rooms[roomId] = &newRoom
    return nil
}

func (app *App) JoinRoom(roomId string, newUser *User) error {
    for _, room := range app.Rooms {
        for _, user := range room.Users {
            if user.Channel == newUser.Channel {
                return &RoomError{Msg: "already joined room", Code: 1}
            }
        }
    }
    app.Rooms[roomId].Users = append(app.Rooms[roomId].Users, *newUser)
    return nil
}

func (room Room) GetInfo() string {
    return strconv.Itoa(len(room.Users)) + "人"
}

func (app *App) TextHandler(ev *slack.MessageEvent) {
    userid := ev.Msg.User
    channel := ev.Msg.Channel
    text := ev.Msg.Text
    if channel[0] != "D"[0] { // if channel[0] == "D": its Direct Message
        return
    }
    user, err := app.Api.GetUserInfo(userid)
    if err != nil {
        fmt.Printf("%s\n", err)
        return
    }
    fmt.Printf("ID: %s, Fullname: %s, Email: %s\n", user.ID, user.Name, user.Profile.Email)
    jinroUser := &User{
        Name: user.Name,
        Channel: channel,
        Job: 0,
        IsAlive: true,
    }
    // username := user.Name
    cmd := CommandParser(text)
    switch cmd[0] {
    case "make":
        if err := app.NewRoom(cmd[1]); err != nil{
            app.SendMessageText("can't maked", channel)
            break
        }
        app.SendMessageText("maked! "+cmd[1], channel)
    case "join":
        if err := app.JoinRoom(cmd[1], jinroUser); err != nil {
            switch e := err.(type) {
            case *RoomError:
                app.SendMessageText(e.Msg, channel)
            default:
                app.SendMessageText("err", channel)
            }
        }
    case "status":
        text := app.Rooms[cmd[1]].GetInfo()
        app.SendMessageText(text, channel)
    case "start":
        app.SendMessageText("huh", channel)
    }
    return
}

func CommandParser(cmd string) []string { // 駄目
    // r := regexp.MustCompile(`^([a-zA-Z][\w-]*)(\s.*)?$`)
    // return r.FindSubmatch(cmd)
    arr := strings.Split(cmd, " ")
    return arr
}

func (app *App) SendMessageText(text, channel string) {
    msg := app.RTM.NewOutgoingMessage(text, channel)
    app.RTM.SendMessage(msg)
    return
}

func NewApp(token string) (*App, error) {
    user := "testuser"
    pass := "testuser"
    host := "localhost"
    dbname := "postgres"

    db, err := sql.Open("postgres", "user="+user+" password="+pass+" host="+host+" dbname="+dbname+" sslmode=disable")
    if err != nil {
        fmt.Printf("fatal error: %v\n", err)
        return nil, err
    }

    api := slack.New(token)
    rtm := api.NewRTM()
    go rtm.ManageConnection()

    app := App{
        Api: api,
        RTM: rtm,
        DB: db,
        Rooms: make(map[string]*Room),
    }
    return &app, nil
}

func main() {
    app, err := NewApp("")
    if err != nil {
        fmt.Printf("%s\n", err)
        return
    }
    logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
    slack.SetLogger(logger)
    app.Api.SetDebug(true)
Loop:
    for {
        select {
        case msg := <-app.RTM.IncomingEvents:
            // fmt.Print("Event Received: ")
            switch ev := msg.Data.(type) {
            case *slack.HelloEvent:
                // Ignore hello

            case *slack.ConnectedEvent:
                fmt.Println("Infos:", ev.Info)
                fmt.Println("Connection counter:", ev.ConnectionCount)
                // Replace #general with your Channel ID
                // rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "#test"))

            case *slack.MessageEvent:
                fmt.Printf("Message: %v\n", ev)
                fmt.Printf("Message: Channnel: %v\nuser: %v\ntext: %v\nName: %v\n", ev.Msg.Channel, ev.Msg.User, ev.Msg.Text, ev.Msg.Name)
                app.TextHandler(ev)

            case *slack.PresenceChangeEvent:
                fmt.Printf("Presence Change: %v\n", ev)

            // case *slack.LatencyReport:
            //     fmt.Printf("Current latency: %v\n", ev.Value)

            case *slack.RTMError:
                fmt.Printf("Error: %s\n", ev.Error())

            case *slack.InvalidAuthEvent:
                fmt.Printf("Invalid credentials")
                break Loop

            default:
                // Ignore other events..
                fmt.Printf("Unexpected: %v\n", msg.Data)
            }
        }
    }
}
