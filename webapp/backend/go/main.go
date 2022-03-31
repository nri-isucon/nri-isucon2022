package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/wesovilabs/koazee"
)

var (
	db                  *sqlx.DB
	mySQLConnectionData *MySQLConnectionEnv
)

type InitializeRequest struct {
	ReservableDays int `json:"reservable_days"`
}

type InitializeResponse struct {
	Language string `json:"language"`
}

type User struct {
	Id        int     `db:"id" json:"id"`
	Password  *string `db:"password" json:"password"`
	Nickname  *string `db:"nickname" json:"nickname"`
	Thumbnail *string `db:"thumbnail" json:"thumbnail"`
}

type Home struct {
	Id           string   `db:"id" json:"id"`
	Name         *string  `db:"name" json:"name"`
	Address      *string  `db:"address" json:"address"`
	Location     *string  `db:"location" json:"location"`
	MaxPeopleNum *int     `db:"max_people_num" json:"max_people_num"`
	Description  *string  `db:"description" json:"description"`
	CatchPhrase  *string  `db:"catch_phrase" json:"catch_phrase"`
	Attribute    *string  `db:"attribute" json:"attribute"`
	Style        *string  `db:"style" json:"style"`
	Price        *int     `db:"price" json:"price"`
	Photo1       *string  `db:"photo_1" json:"photo_1"`
	Photo2       *string  `db:"photo_2" json:"photo_2"`
	Photo3       *string  `db:"photo_3" json:"photo_3"`
	Photo4       *string  `db:"photo_4" json:"photo_4"`
	Photo5       *string  `db:"photo_5" json:"photo_5"`
	Rate         *float64 `db:"rate" json:"rate"`
	OwnerId      string   `db:"owner_id" json:"owner_id"`
}

type HomesResponse struct {
	Count int    `json:"count"`
	Homes []Home `json:"homes"`
}

type PostHomesResponse struct {
	Count int `json:"count"`
}

type ReservationHome struct {
	Id             string          `db:"id" json:"id"`
	UserId         int             `db:"user_id" json:"user_id"`
	HomeId         int             `db:"home_id" json:"home_id"`
	Date           *string         `db:"date" json:"date"`
	NumberOfPeople *int            `db:"number_of_people" json:"number_of_people"`
	IsDeleted      *sql.NullString `db:"is_deleted" json:"is_deleted"`
}

type ReservationHomeInfo struct {
	ReservationId  string `db:"reservation_id" json:"reservation_id"`
	NumberOfPeople *int   `db:"number_of_people" json:"number_of_people"`
	HomeId         string `db:"home_id" json:"home_id"`
}

type ReservationHomeRequest struct {
	HomeId         *string `json:"home_id"`
	UserId         *string `json:"user_id"`
	StartDate      *string `json:"start_date"`
	EndDate        *string `json:"end_date"`
	NumberOfPeople *int    `json:"number_of_people"`
}

type ReservationActivityRequest struct {
	ActivityId     *string `json:"activity_id"`
	UserId         *string `json:"user_id"`
	Date           *string `json:"date"`
	NumberOfPeople *int    `json:"number_of_people"`
}

type ReservationHomeResponse struct {
	Result bool `json:"result"`
}

type ReservationActivityResponse struct {
	Result bool `json:"result"`
}

type UserReservationHomeResponse struct {
	Reservations []UserReservationHome `json:"reservations"`
}

type UserReservationHome struct {
	ReserveId      string `json:"reserve_id"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	NumberOfPeople *int   `json:"number_of_people"`
	ReserveHome    Home   `json:"reserve_home"`
}

type Activity struct {
	Id           string   `db:"id" json:"id"`
	Name         *string  `db:"name" json:"name"`
	Address      *string  `db:"address" json:"address"`
	Location     *string  `db:"location" json:"location"`
	MaxPeopleNum *int     `db:"max_people_num" json:"max_people_num"`
	Description  *string  `db:"description" json:"description"`
	CatchPhrase  *string  `db:"catch_phrase" json:"catch_phrase"`
	Attribute    *string  `db:"attribute" json:"attribute"`
	Category     *string  `db:"category" json:"category"`
	Price        *int     `db:"price" json:"price"`
	Photo1       *string  `db:"photo_1" json:"photo_1"`
	Photo2       *string  `db:"photo_2" json:"photo_2"`
	Photo3       *string  `db:"photo_3" json:"photo_3"`
	Photo4       *string  `db:"photo_4" json:"photo_4"`
	Photo5       *string  `db:"photo_5" json:"photo_5"`
	Rate         *float64 `db:"rate" json:"rate"`
	OwnerId      string   `db:"owner_id" json:"owner_id"`
}

type ActivitiesResponse struct {
	Count      int        `json:"count"`
	Activities []Activity `json:"activities"`
}

type ReservationActivity struct {
	Id             string          `db:"id" json:"id"`
	UserId         int             `db:"user_id" json:"user_id"`
	ActivityId     int             `db:"activity_id" json:"activity_id"`
	Date           *string         `db:"date" json:"date"`
	NumberOfPeople *int            `db:"number_of_people" json:"number_of_people"`
	IsDeleted      *sql.NullString `db:"is_deleted" json:"is_deleted"`
}

type ReservationActivityInfo struct {
	ReservationId  string `db:"reservation_id" json:"reservation_id"`
	NumberOfPeople *int   `db:"number_of_people" json:"number_of_people"`
	ActivityId     string `db:"activity_id" json:"activity_id"`
}

type UserReservationActivityResponse struct {
	Reservations []UserReservationActivity `json:"reservations"`
}

type UserReservationActivity struct {
	ReserveId       string   `json:"reserve_id"`
	ReserveDate     string   `json:"reserve_date"`
	NumberOfPeople  *int     `json:"number_of_people"`
	ReserveActivity Activity `json:"reserve_activity"`
}

type CalenderResponse struct {
	Items []IsReservable `json:"items"`
}

type IsReservable struct {
	Date      string `json:"date"`
	Available bool   `json:"available"`
}

type MySQLConnectionEnv struct {
	Host     string
	Port     string
	User     string
	DBName   string
	Password string
}

type RecordMapper struct {
	Record []string

	offset int
	err    error
}

func (r *RecordMapper) next() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	if r.offset >= len(r.Record) {
		r.err = fmt.Errorf("too many read")
		return "", r.err
	}
	s := r.Record[r.offset]
	r.offset++
	return s, nil
}

func (r *RecordMapper) NextInt() int {
	s, err := r.next()
	if err != nil {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		r.err = err
		return 0
	}
	return i
}

func (r *RecordMapper) NextFloat() float64 {
	s, err := r.next()
	if err != nil {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		r.err = err
		return 0
	}
	return f
}

func (r *RecordMapper) NextString() string {
	s, err := r.next()
	if err != nil {
		return ""
	}
	return s
}

func (r *RecordMapper) Err() error {
	return r.err
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func NewMySQLConnectionEnv() *MySQLConnectionEnv {
	return &MySQLConnectionEnv{
		Host:     getEnv("MYSQL_HOST", "127.0.0.1"),
		Port:     getEnv("MYSQL_PORT", "3306"),
		User:     getEnv("MYSQL_USER", "isucon"),
		DBName:   getEnv("MYSQL_DBNAME", "isubnb"),
		Password: getEnv("MYSQL_PASS", "isucon"),
	}
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultValue
}

//ConnectDB isubnbデータベースに接続する
func (mc *MySQLConnectionEnv) ConnectDB() (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?parseTime=true&loc=Asia%%2FTokyo", mc.User, mc.Password, mc.Host, mc.Port, mc.DBName)
	return sqlx.Open("mysql", dsn)
}

func convertToResponseHome(home Home) Home {
	imagePath1 := "/api/v1/home/" + home.Id + "/image/1"
	imagePath2 := "/api/v1/home/" + home.Id + "/image/2"
	imagePath3 := "/api/v1/home/" + home.Id + "/image/3"
	imagePath4 := "/api/v1/home/" + home.Id + "/image/4"
	imagePath5 := "/api/v1/home/" + home.Id + "/image/5"
	home.Photo1 = &imagePath1
	home.Photo2 = &imagePath2
	home.Photo3 = &imagePath3
	home.Photo4 = &imagePath4
	home.Photo5 = &imagePath5
	return home
}

func convertToResponseActivity(activity Activity) Activity {
	imagePath1 := "/api/v1/activity/" + activity.Id + "/image/1"
	imagePath2 := "/api/v1/activity/" + activity.Id + "/image/2"
	imagePath3 := "/api/v1/activity/" + activity.Id + "/image/3"
	imagePath4 := "/api/v1/activity/" + activity.Id + "/image/4"
	imagePath5 := "/api/v1/activity/" + activity.Id + "/image/5"
	activity.Photo1 = &imagePath1
	activity.Photo2 = &imagePath2
	activity.Photo3 = &imagePath3
	activity.Photo4 = &imagePath4
	activity.Photo5 = &imagePath5
	return activity
}

func main() {
	// Echo instance
	e := echo.New()
	e.Debug = true
	e.Logger.SetLevel(log.DEBUG)

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Initialize
	e.POST("/api/v1/initialize", initialize)

	// Home Handler
	e.GET("/api/v1/homes", getApiV1Homes)
	e.POST("/api/v1/homes", postApiV1Homes)
	e.GET("/api/v1/home/:homeId", getApiV1Home)
	e.GET("/api/v1/home/:homeId/image/:imageId", getApiV1HomeImage)
	e.GET("/api/v1/home/:homeId/calendar", getApiV1HomeCalendar)
	e.POST("/api/v1/reservation_home", postApiV1ReservationHome)
	e.GET("/api/v1/user/:userId/reservation_home", getApiV1UserReservationHome)
	e.DELETE("/api/v1/reservation_home/:reservationHomeId", deleteApiV1ReservationHome)

	// Activity Handler
	e.GET("/api/v1/activities", getApiV1Activities)
	e.GET("/api/v1/activity/:activityId", getApiV1Activity)
	e.GET("/api/v1/activity/:activityId/image/:imageId", getApiV1ActivityImage)
	e.POST("/api/v1/reservation_activity", postApiV1ReservationActivity)
	e.GET("/api/v1/user/:userId/reservation_activity", getApiV1UserReservationActivity)
	e.DELETE("/api/v1/reservation_activity/:reservationActivityId", deleteApiV1ReservationActivity)

	mySQLConnectionData = NewMySQLConnectionEnv()

	var err error
	db, err = mySQLConnectionData.ConnectDB()
	if err != nil {
		e.Logger.Fatalf("failed to connect db: %v", err)
		return
	}
	db.SetMaxOpenConns(10)
	defer db.Close()

	// Start server
	serverPort := fmt.Sprintf(":%v", getEnv("SERVER_PORT", "1323"))
	e.Logger.Fatal(e.Start(serverPort))
}

func initialize(c echo.Context) error {
	var request InitializeRequest
	err := c.Bind(&request)
	if err != nil {
		c.Logger().Errorf("Request error : %v", err)
		return c.String(http.StatusBadRequest, "Bad request body.")
	}
	c.Logger().Infof("Request reservableDays=[%v]", request.ReservableDays)

	sqlDir := "/home/isucon/isubnb/webapp/backend/mysql/db/"
	paths := []string{
		filepath.Join(sqlDir, "0_Schema.sql"),
		filepath.Join(sqlDir, "1_CsvDataImport.sql"),
	}

	for _, p := range paths {
		sqlFile, _ := filepath.Abs(p)
		cmdStr := fmt.Sprintf("mysql -h %v -u %v -p%v -P %v %v < %v",
			mySQLConnectionData.Host,
			mySQLConnectionData.User,
			mySQLConnectionData.Password,
			mySQLConnectionData.Port,
			mySQLConnectionData.DBName,
			sqlFile,
		)
		cmd := exec.Command("bash", "-c", cmdStr)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stderr
		err = cmd.Run()
		if err != nil {
			c.Logger().Errorf("exec error: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	_, err = db.Exec(
		"INSERT INTO `config` (`reservable_days`) VALUES (?)",
		request.ReservableDays,
	)
	if err != nil {
		c.Logger().Errorf("db error : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, InitializeResponse{
		Language: "go",
	})
}

func getApiV1Homes(c echo.Context) error {

	var homesResponse HomesResponse
	homesResponse.Homes = []Home{}

	location := c.QueryParam("location")
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")
	numberOfPeople := c.QueryParam("number_of_people")
	style := c.QueryParam("style")

	getAllHomesQuery := `SELECT * FROM isubnb.home ORDER BY rate DESC, price ASC, name ASC`
	err := db.Select(&homesResponse.Homes, getAllHomesQuery)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if startDate != "" && endDate != "" {
		var matchedHome []Home
		for _, home := range homesResponse.Homes {
			var reservationHome []ReservationHome
			getIsReserveHomeQuery := `SELECT * FROM isubnb.reservation_home WHERE home_id = ? AND ? <= date AND date < ?`
			err := db.Select(&reservationHome, getIsReserveHomeQuery, home.Id, startDate, endDate)
			if err != nil {
				c.Echo().Logger.Errorf("Error occurred : %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
			if len(reservationHome) == 0 {
				matchedHome = append(matchedHome, home)
			}
		}
		homesResponse.Homes = matchedHome
	}

	if location != "" {
		homesResponse.Homes = koazee.StreamOf(homesResponse.Homes).Filter(func(home Home) bool {
			return *home.Location == location
		}).Out().Val().([]Home)
	}
	if numberOfPeople != "" {
		numberOfPeopleInt, err := strconv.Atoi(numberOfPeople)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusBadRequest)
		}
		homesResponse.Homes = koazee.StreamOf(homesResponse.Homes).Filter(func(home Home) bool {
			return *home.MaxPeopleNum >= numberOfPeopleInt
		}).Out().Val().([]Home)
	}
	if style != "" {
		homesResponse.Homes = koazee.StreamOf(homesResponse.Homes).Filter(func(home Home) bool {
			return *home.Style == style
		}).Out().Val().([]Home)
	}

	homesResponse.Homes = koazee.StreamOf(homesResponse.Homes).Map(func(home Home) Home {
		return convertToResponseHome(home)
	}).Out().Val().([]Home)

	homesResponse.Count = len(homesResponse.Homes)
	return c.JSON(http.StatusOK, homesResponse)
}

func postApiV1Homes(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		c.Logger().Errorf("failed to get form file: %v", err)
		return err
	}

	headers := form.File["homes.csv"]
	if len(headers) == 0 {
		c.Logger().Errorf("csv file missing.")
		response := ErrorResponse{
			Message: "正しい名前のCSVファイルを送信してください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	f, err := headers[0].Open()
	if err != nil {
		return err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		c.Logger().Errorf("failed to read csv: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	tx, err := db.Begin()
	if err != nil {
		c.Logger().Errorf("failed to begin tx: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer tx.Rollback()

	var counter = 0
	for _, row := range records {
		rm := RecordMapper{Record: row}
		id := rm.NextString()
		name := rm.NextString()
		address := rm.NextString()
		location := rm.NextString()
		maxPeopleNum := rm.NextInt()
		description := rm.NextString()
		catchPhrase := rm.NextString()
		attribute := rm.NextString()
		style := rm.NextString()
		price := rm.NextInt()
		photo1 := rm.NextString()
		photo2 := rm.NextString()
		photo3 := rm.NextString()
		photo4 := rm.NextString()
		photo5 := rm.NextString()
		rate := rm.NextFloat()
		ownerId := rm.NextString()
		if err := rm.Err(); err != nil {
			c.Logger().Errorf("failed to read record: %v", err)
			return c.NoContent(http.StatusBadRequest)
		}
		_, err := tx.Exec("INSERT INTO isubnb.home(id, name, address, location, max_people_num, description, catch_phrase, attribute, style, price, photo_1, photo_2, photo_3, photo_4, photo_5, rate, owner_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, name, address, location, maxPeopleNum, description, catchPhrase, attribute, style, price, photo1, photo2, photo3, photo4, photo5, rate, ownerId)
		if err != nil {
			c.Logger().Errorf("failed to insert home: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		counter++
	}
	if err := tx.Commit(); err != nil {
		c.Logger().Errorf("failed to commit tx: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	response := PostHomesResponse{
		Count: counter,
	}
	return c.JSON(http.StatusOK, response)
}

func getApiV1Home(c echo.Context) error {
	homeId := c.Param("homeId")

	home := Home{}
	var homeList []Home

	// 宿確認
	getHomeQuery := `SELECT * FROM isubnb.home WHERE id = ?`
	err := db.Select(&homeList, getHomeQuery, homeId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(homeList) == 0 {
		c.Echo().Logger.Error("対象宿が存在しません。")
		response := ErrorResponse{
			Message: "対象宿が存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	err = db.Select(&homeList, getHomeQuery, homeId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	home = homeList[0]
	home = convertToResponseHome(home)

	return c.JSON(http.StatusOK, home)
}

func getApiV1HomeImage(c echo.Context) error {
	homeId := c.Param("homeId")

	var homeList []Home
	getHomeQuery := `SELECT * FROM isubnb.home WHERE id = ?`
	err := db.Select(&homeList, getHomeQuery, homeId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(homeList) == 0 {
		c.Echo().Logger.Error("対象宿が存在しません。")
		response := ErrorResponse{
			Message: "対象宿が存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	result := map[int]*string{}
	result[0] = homeList[0].Photo1
	result[1] = homeList[0].Photo2
	result[2] = homeList[0].Photo3
	result[3] = homeList[0].Photo4
	result[4] = homeList[0].Photo5

	imageId, err := strconv.Atoi(c.Param("imageId"))
	if err != nil || imageId < 1 || imageId > 5 {
		c.Echo().Logger.Error("画像IDの指定が誤っています。")
		response := ErrorResponse{
			Message: "画像IDの指定が誤っています。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	if result[imageId-1] == nil {
		response := ErrorResponse{
			Message: "画像が存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	dec, err := base64.StdEncoding.DecodeString(*result[imageId-1])
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.Blob(http.StatusOK, "image/jpeg", dec)
}

func getApiV1HomeCalendar(c echo.Context) error {
	homeId := c.Param("homeId")

	var homeList []Home
	var reservableDays []int
	getConfigQuery := `SELECT * FROM isubnb.config`
	err := db.Select(&reservableDays, getConfigQuery)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if reservableDays[0] == 0 {
		response := ErrorResponse{
			Message: "予約可能日数が0日です。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// 宿確認
	getHomeQuery := `SELECT * FROM isubnb.home WHERE id = ?`
	err = db.Select(&homeList, getHomeQuery, homeId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(homeList) == 0 {
		response := ErrorResponse{
			Message: "対象宿が存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	date := time.Now()
	endDate := date.AddDate(0, 0, reservableDays[0])

	var calenderList CalenderResponse
	for endDate.Sub(date).Hours() >= 24 {
		var reservationHomeId []ReservationHome
		formatDate := date.Format("2006-01-02")
		getReservationHomeQuery := `SELECT * FROM isubnb.reservation_home WHERE home_id = ? AND DATE(date) = ? AND is_deleted = ? ORDER BY user_id, home_id`
		err = db.Select(&reservationHomeId, getReservationHomeQuery, homeId, formatDate, 0)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		var isReservable IsReservable
		isReservable.Date = formatDate
		if len(reservationHomeId) == 0 {
			isReservable.Available = true
		} else {
			isReservable.Available = false
		}
		calenderList.Items = append(calenderList.Items, isReservable)
		date = date.AddDate(0, 0, 1)
	}

	return c.JSON(http.StatusOK, calenderList)
}

func postApiV1ReservationHome(c echo.Context) error {
	var request ReservationHomeRequest
	err := c.Bind(&request)
	if err != nil {
		c.Logger().Errorf("Request error : %v", err)
		return c.String(http.StatusBadRequest, "Bad request body.")
	}

	if request.UserId == nil {
		response := ErrorResponse{
			Message: "ユーザIDを入力してください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	var user []User
	getUserQuery := `SELECT * FROM isubnb.user WHERE id = ?`
	err = db.Select(&user, getUserQuery, request.UserId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(user) == 0 {
		response := ErrorResponse{
			Message: "対象ユーザが存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	if request.HomeId == nil {
		response := ErrorResponse{
			Message: "宿IDを入力してください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	var home []Home
	getHomeQuery := `SELECT * FROM isubnb.home WHERE id = ?`
	err = db.Select(&home, getHomeQuery, request.HomeId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(home) == 0 {
		response := ErrorResponse{
			Message: "対象宿が存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	err = db.Select(&home, getHomeQuery, request.HomeId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if *request.NumberOfPeople > *home[0].MaxPeopleNum {
		response := ErrorResponse{
			Message: "予約可能人数を超えています。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}
	r, _ := regexp.Compile("^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$")
	if !r.MatchString(*request.StartDate) || !r.MatchString(*request.EndDate) {
		response := ErrorResponse{
			Message: "日付はyyyy-mm-dd形式で入力してください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	startDate, _ := time.Parse("2006-01-02", *request.StartDate)
	endDate, _ := time.Parse("2006-01-02", *request.EndDate)
	if endDate.Sub(startDate).Hours() < 24 {
		response := ErrorResponse{
			Message: "日付間隔を1日以上にしてください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// 予約確認
	var reservationHomeIdList []string
	getReservationHomeQuery := `SELECT rh.id FROM isubnb.home h JOIN isubnb.reservation_home rh ON h.id=rh.home_id WHERE h.id = ? AND ? <= rh.date AND rh.date < ? AND rh.is_deleted = ? `
	err = db.Select(&reservationHomeIdList, getReservationHomeQuery, request.HomeId, request.StartDate, request.EndDate, 0)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(reservationHomeIdList) != 0 {
		response := ErrorResponse{
			Message: "既に予約が入っているため、予約できません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// 予約
	reserveId, _ := uuid.NewRandom()
	reserveIdStr := reserveId.String()
	for num := 0; num < int(endDate.Sub(startDate).Hours()/24); num++ {
		saveReservationHomeQuery := `INSERT INTO isubnb.reservation_home(id, user_id, home_id, date, number_of_people, is_deleted) VALUES (?, ?, ?, ?, ?, ?)`
		_ = db.MustExec(saveReservationHomeQuery, reserveIdStr, request.UserId, request.HomeId, startDate.AddDate(0, 0, num), *request.NumberOfPeople, false)
	}

	var response ReservationHomeResponse
	response.Result = true

	return c.JSON(http.StatusOK, response)
}

func getApiV1UserReservationHome(c echo.Context) error {
	userId := c.Param("userId")

	var user []User
	getUserQuery := `SELECT * FROM isubnb.user WHERE id = ?`
	err := db.Select(&user, getUserQuery, userId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(user) == 0 {
		response := ErrorResponse{
			Message: "対象ユーザが存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	var reservationHomeList []ReservationHomeInfo
	getReservationHomeQuery := `SELECT DISTINCT rh.id as reservation_id, rh.number_of_people, rh.home_id FROM isubnb.user u JOIN isubnb.reservation_home rh ON u.id = rh.user_id WHERE u.id = ? AND rh.is_deleted = ?`
	err = db.Select(&reservationHomeList, getReservationHomeQuery, userId, 0)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response UserReservationHomeResponse
	response.Reservations = []UserReservationHome{}
	for _, reservationHome := range reservationHomeList {
		var homeList []Home
		getHomeQuery := `SELECT * FROM isubnb.home WHERE id = ?`
		err = db.Select(&homeList, getHomeQuery, reservationHome.HomeId)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		home := convertToResponseHome(homeList[0])

		reserveId := reservationHome.ReservationId
		numberOfPeople := reservationHome.NumberOfPeople

		var startDateTime []time.Time
		getStartDateQuery := `SELECT min(date) FROM isubnb.reservation_home WHERE id = ?`
		err := db.Select(&startDateTime, getStartDateQuery, reserveId)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		startDate := startDateTime[0].Format("2006-01-02")

		var endDateTime []time.Time
		getEndDateQuery := `SELECT max(date) FROM isubnb.reservation_home WHERE id = ?`
		err = db.Select(&endDateTime, getEndDateQuery, reserveId)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		endDate := endDateTime[0].AddDate(0, 0, 1).Format("2006-01-02")

		var userReservationHome UserReservationHome
		userReservationHome.ReserveId = reserveId
		userReservationHome.StartDate = startDate
		userReservationHome.EndDate = endDate
		userReservationHome.NumberOfPeople = numberOfPeople
		userReservationHome.ReserveHome = home

		response.Reservations = append(response.Reservations, userReservationHome)
	}

	return c.JSON(http.StatusOK, response)
}

func deleteApiV1ReservationHome(c echo.Context) error {
	reservationHomeId := c.Param("reservationHomeId")

	var reservationHome []ReservationHome
	getReservationHomeQuery := `SELECT * FROM isubnb.reservation_home WHERE id = ? AND is_deleted = ?`
	err := db.Select(&reservationHome, getReservationHomeQuery, reservationHomeId, 0)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(reservationHome) == 0 {
		response := ErrorResponse{
			Message: "対象の予約が存在しませんでした。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	deleteReservationHomeQuery := `UPDATE isubnb.reservation_home SET is_deleted = ? WHERE id = ?`
	result := db.MustExec(deleteReservationHomeQuery, 1, reservationHomeId)
	_, err = result.RowsAffected()
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response ReservationHomeResponse
	response.Result = true
	return c.JSON(http.StatusOK, response)
}

func getApiV1Activities(c echo.Context) error {
	var activitiesResponse ActivitiesResponse
	activitiesResponse.Activities = []Activity{}

	location := c.QueryParam("location")
	date := c.QueryParam("date")

	getAllActivitiesQuery := `SELECT * FROM isubnb.activity ORDER BY rate DESC, price ASC, name ASC`
	err := db.Select(&activitiesResponse.Activities, getAllActivitiesQuery)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if location != "" {
		activitiesResponse.Activities = koazee.StreamOf(activitiesResponse.Activities).Filter(func(activities Activity) bool {
			return *activities.Location == location
		}).Out().Val().([]Activity)
	}

	if date != "" {
		var matchedActivities []Activity
		for _, activity := range activitiesResponse.Activities {
			var reservationActivity []ReservationActivity
			getIsReserveActivityQuery := `SELECT * FROM isubnb.reservation_activity WHERE activity_id = ? AND date = ?`
			err := db.Select(&reservationActivity, getIsReserveActivityQuery, activity.Id, date)
			if err != nil {
				c.Echo().Logger.Errorf("Error occurred : %v", err)
				return c.NoContent(http.StatusInternalServerError)
			}
			if len(reservationActivity) == 0 {
				matchedActivities = append(matchedActivities, activity)
			}
		}
		activitiesResponse.Activities = matchedActivities
	}

	activitiesResponse.Activities = koazee.StreamOf(activitiesResponse.Activities).Map(func(activity Activity) Activity {
		return convertToResponseActivity(activity)
	}).Out().Val().([]Activity)

	activitiesResponse.Count = len(activitiesResponse.Activities)
	return c.JSON(http.StatusOK, activitiesResponse)
}

func getApiV1Activity(c echo.Context) error {
	activityId := c.Param("activityId")

	activity := Activity{}
	var activityList []Activity

	getActivityQuery := `SELECT * FROM isubnb.activity WHERE id = ?`
	err := db.Select(&activityList, getActivityQuery, activityId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(activityList) == 0 {
		response := ErrorResponse{
			Message: "対象アクティビティが存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	err = db.Select(&activityList, getActivityQuery, activityId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	activity = activityList[0]
	activity = convertToResponseActivity(activity)

	return c.JSON(http.StatusOK, activity)
}

func getApiV1ActivityImage(c echo.Context) error {
	activityId := c.Param("activityId")

	var activityList []Activity
	getActivityQuery := `SELECT * FROM isubnb.activity WHERE id = ?`
	err := db.Select(&activityList, getActivityQuery, activityId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(activityList) == 0 {
		response := ErrorResponse{
			Message: "対象アクティビティが存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	result := map[int]*string{}
	result[0] = activityList[0].Photo1
	result[1] = activityList[0].Photo2
	result[2] = activityList[0].Photo3
	result[3] = activityList[0].Photo4
	result[4] = activityList[0].Photo5

	imageId, err := strconv.Atoi(c.Param("imageId"))
	if err != nil || imageId < 1 || imageId > 5 {
		response := ErrorResponse{
			Message: "画像IDの指定が誤っています。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	if result[imageId-1] == nil {
		response := ErrorResponse{
			Message: "画像が存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	dec, err := base64.StdEncoding.DecodeString(*result[imageId-1])
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.Blob(http.StatusOK, "image/jpeg", dec)
}

func postApiV1ReservationActivity(c echo.Context) error {
	var request ReservationActivityRequest
	err := c.Bind(&request)
	if err != nil {
		c.Logger().Errorf("Request error : %v", err)
		return c.String(http.StatusBadRequest, "Bad request body.")
	}

	if request.UserId == nil {
		response := ErrorResponse{
			Message: "ユーザIDを入力してください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	var user []User
	getUserQuery := `SELECT * FROM isubnb.user WHERE id = ?`
	err = db.Select(&user, getUserQuery, request.UserId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(user) == 0 {
		response := ErrorResponse{
			Message: "対象ユーザが存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	if request.ActivityId == nil {
		response := ErrorResponse{
			Message: "アクティビティIDを入力してください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	var activity []Activity
	getActivityQuery := `SELECT * FROM isubnb.activity WHERE id = ?`
	err = db.Select(&activity, getActivityQuery, request.ActivityId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(activity) == 0 {
		response := ErrorResponse{
			Message: "対象アクティビティが存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	err = db.Select(&activity, getActivityQuery, request.ActivityId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if *request.NumberOfPeople > *activity[0].MaxPeopleNum {
		response := ErrorResponse{
			Message: "予約可能人数を超えています。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}
	r, _ := regexp.Compile("^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$")
	if !r.MatchString(*request.Date) {
		response := ErrorResponse{
			Message: "日付はyyyy-mm-dd形式で入力してください。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	date, _ := time.Parse("2006-01-02", *request.Date)

	// 予約確認
	var reservationActivityIdList []string
	getReservationActivityQuery := `SELECT ra.id FROM isubnb.activity a JOIN isubnb.reservation_activity ra ON a.id = ra.activity_id WHERE a.id = ? AND ra.date = ? AND ra.is_deleted = ? `
	err = db.Select(&reservationActivityIdList, getReservationActivityQuery, request.ActivityId, request.Date, 0)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(reservationActivityIdList) != 0 {
		response := ErrorResponse{
			Message: "既に予約が入っているため、予約できません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	// 予約
	reserveId, _ := uuid.NewRandom()
	reserveIdStr := reserveId.String()
	saveReservationActivityQuery := `INSERT INTO isubnb.reservation_activity(id, user_id, activity_id, date, number_of_people, is_deleted) VALUES (?, ?, ?, ?, ?, ?)`
	_ = db.MustExec(saveReservationActivityQuery, reserveIdStr, request.UserId, request.ActivityId, date, *request.NumberOfPeople, false)

	var response ReservationActivityResponse
	response.Result = true

	return c.JSON(http.StatusOK, response)
}

func getApiV1UserReservationActivity(c echo.Context) error {
	userId := c.Param("userId")

	var user []User
	getUserQuery := `SELECT * FROM isubnb.user WHERE id = ?`
	err := db.Select(&user, getUserQuery, userId)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(user) == 0 {
		response := ErrorResponse{
			Message: "対象ユーザが存在しません。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	var reservationActivityList []ReservationActivityInfo
	getReservationActivityQuery := `SELECT DISTINCT ra.id as reservation_id, ra.number_of_people, ra.activity_id FROM isubnb.user u JOIN isubnb.reservation_activity ra ON u.id = ra.user_id WHERE u.id = ? AND ra.is_deleted = ?`
	err = db.Select(&reservationActivityList, getReservationActivityQuery, userId, 0)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response UserReservationActivityResponse
	response.Reservations = []UserReservationActivity{}
	for _, reservationActivity := range reservationActivityList {
		var activityList []Activity
		getActivityQuery := `SELECT * FROM isubnb.activity WHERE id = ?`
		err = db.Select(&activityList, getActivityQuery, reservationActivity.ActivityId)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		activity := convertToResponseActivity(activityList[0])

		reserveId := reservationActivity.ReservationId
		numberOfPeople := reservationActivity.NumberOfPeople

		var startDateTime []time.Time
		getStartDateQuery := `SELECT min(date) FROM isubnb.reservation_activity WHERE id = ?`
		err := db.Select(&startDateTime, getStartDateQuery, reserveId)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		startDate := startDateTime[0].Format("2006-01-02")

		var endDateTime []time.Time
		getEndDateQuery := `SELECT max(date) FROM isubnb.reservation_activity WHERE id = ?`
		err = db.Select(&endDateTime, getEndDateQuery, reserveId)
		if err != nil {
			c.Echo().Logger.Errorf("Error occurred : %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		_ = endDateTime[0].AddDate(0, 0, 1).Format("2006-01-02")

		var userReservationActivity UserReservationActivity
		userReservationActivity.ReserveId = reserveId
		userReservationActivity.ReserveDate = startDate
		userReservationActivity.NumberOfPeople = numberOfPeople
		userReservationActivity.ReserveActivity = activity

		response.Reservations = append(response.Reservations, userReservationActivity)
	}

	return c.JSON(http.StatusOK, response)
}

func deleteApiV1ReservationActivity(c echo.Context) error {
	reservationActivityId := c.Param("reservationActivityId")

	var reservationActivity []ReservationActivity
	getReservationActivityQuery := `SELECT * FROM isubnb.reservation_activity WHERE id = ? AND is_deleted = ?`
	err := db.Select(&reservationActivity, getReservationActivityQuery, reservationActivityId, 0)
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if len(reservationActivity) == 0 {
		response := ErrorResponse{
			Message: "対象の予約が存在しませんでした。",
		}
		return c.JSON(http.StatusBadRequest, response)
	}

	deleteReservationActivityQuery := `UPDATE isubnb.reservation_activity SET is_deleted = ? WHERE id = ?`
	result := db.MustExec(deleteReservationActivityQuery, 1, reservationActivityId)
	_, err = result.RowsAffected()
	if err != nil {
		c.Echo().Logger.Errorf("Error occurred : %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var response ReservationHomeResponse
	response.Result = true
	return c.JSON(http.StatusOK, response)
}
