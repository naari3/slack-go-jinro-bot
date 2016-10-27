package main

import (
    "fmt"
    "log"
    "os"
    // "reflect"
    "sync"
    _ "github.com/lib/pq"
    "database/sql"

    "github.com/nlopes/slack"
)

type App struct{
    Api *slack.Client
    RTM *slack.RTM
    DB *sql.DB
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
    }
    return &app, nil
}

func (app *App) StartGame(ju *JinroUser) {
    fmt.Printf("jinrouser's channel: %v\n", ju.Channel)
}

type Rooms struct{
    Rooms []Room
}

type Room struct{
    JinroUsers []JinroUser
    Status int
    mux sync.Mutex
}

func (r *Room) GetRoomById(roomId string) *Room {
    r.mux.Lock()
    defer r.mux.Unlock()
    return r
}

type JinroUser struct{
    User *slack.User
    Channel string
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
    // username := user.Name
    switch text {
    case "help":
        text := "plz say \"start\""
        msg := app.RTM.NewOutgoingMessage(text, channel)
        app.RTM.SendMessage(msg)
    case "start":
        ju := JinroUser{
            User: user,
            Channel: channel,
        }
        app.StartGame(&ju)
    }
    return
}

func main() {
    app, err := NewApp("token")
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
