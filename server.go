package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Person struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"column:name"`
	Surname     string `gorm:"column:surname"`
	Patronymic  string `gorm:"column:patronymic"`
	Gender      string `gorm:"column:gender"`
	Age         int    `gorm:"column:age"`
	Nationality string `gorm:"column:nationality"`
}

type App struct {
	DB *gorm.DB
}

func (app *App) Initialize() {
	/*
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}

		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			godotenv.Load("DB_HOST"),
			godotenv.Load("DB_USER"),
			godotenv.Load("DB_PASSWORD"),
			godotenv.Load("DB_NAME"),
			godotenv.Load("DB_PORT"),
			godotenv.Load("DB_SSL_MODE"),
		)
	*/

	dsn := "host=db user=postgres password=XDDDPASSW0RD dbname=EFMOBILPersons port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	app.DB = db

	// Автоматическая миграция в БД
	app.DB.AutoMigrate(&Person{})
}

func (app *App) MainPage(c *fiber.Ctx) error {
	c.Status(http.StatusOK)
	return c.SendString("Тут ничего нет.")
}

func GetAge(name string) (int, error) {
	ageAPIURL := fmt.Sprintf("https://api.agify.io/?name=%s", name)
	ageResponse, err := http.Get(ageAPIURL)
	if err != nil {
		return 0, err
	}
	defer ageResponse.Body.Close()

	ageData, err := io.ReadAll(ageResponse.Body)
	if err != nil {
		return 0, err
	}

	var ageResult struct {
		Age int `json:"age"`
	}
	err = json.Unmarshal(ageData, &ageResult)
	if err != nil {
		return 0, err
	}

	return ageResult.Age, nil
}

func GetNationality(name string) (string, error) {
	nationalityAPIURL := fmt.Sprintf("https://api.nationalize.io/?name=%s", name)
	nationalityResponse, err := http.Get(nationalityAPIURL)
	if err != nil {
		return "", err
	}
	defer nationalityResponse.Body.Close()

	nationalityData, err := io.ReadAll(nationalityResponse.Body)
	if err != nil {
		return "", err
	}

	var nationalityResult struct {
		Country []struct {
			CountryID string `json:"country_id"`
		} `json:"country"`
	}
	err = json.Unmarshal(nationalityData, &nationalityResult)
	if err != nil {
		return "", err
	}

	if len(nationalityResult.Country) > 0 {
		return nationalityResult.Country[0].CountryID, nil
	}

	return "", nil
}

func GetGender(name string) (string, error) {
	genderAPIURL := fmt.Sprintf("https://api.genderize.io/?name=%s", name)
	genderResponse, err := http.Get(genderAPIURL)
	if err != nil {
		return "", err
	}
	defer genderResponse.Body.Close()

	genderData, err := io.ReadAll(genderResponse.Body)
	if err != nil {
		return "", err
	}

	var genderResult struct {
		Gender string `json:"gender"`
	}
	err = json.Unmarshal(genderData, &genderResult)
	if err != nil {
		return "", err
	}

	return genderResult.Gender, nil
}

func (app *App) AddPerson(c *fiber.Ctx) error {
	name := c.FormValue("name")
	surname := c.FormValue("surname")
	patronymic := c.FormValue("patronymic")

	// Обогащаем данные о человеке
	age, err := GetAge(name)
	if err != nil {
		return err
	}

	gender, err := GetGender(name)
	if err != nil {
		return err
	}

	nationality, err := GetNationality(name)
	if err != nil {
		return err
	}

	person := Person{
		Name:        name,
		Surname:     surname,
		Patronymic:  patronymic,
		Gender:      gender,
		Age:         age,
		Nationality: nationality,
	}

	result := app.DB.Create(&person)
	if result.Error != nil {
		// Обработка ошибки сохранения в базе данных
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": result.Error.Error(),
		})
	}

	// Возвращаем успешный ответ
	return c.JSON(fiber.Map{
		"message": "Person added successfully",
		"person":  person,
	})

}

func (app *App) DeleteData(c *fiber.Ctx) error {
	uid := c.FormValue("id")
	var person Person

	if err := app.DB.Where("id = ?", uid).Delete(&person).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	app.DB.Delete(&person)
	return c.JSON(fiber.Map{
		"message": "Person deleted successfully",
	})
}

func (app *App) UpdateData(c *fiber.Ctx) error {
	id := c.FormValue("id")
	name := c.FormValue("name")
	surname := c.FormValue("surname")
	patronymic := c.FormValue("patronymic")
	gender := c.FormValue("gender")
	age := c.FormValue("age")
	nationality := c.FormValue("nationality")

	// Находим запись в базе данных по ID
	var person Person
	if err := app.DB.First(&person, id).Error; err != nil {
		return err
	}

	if name != "" {
		person.Name = name
	}

	if surname != "" {
		person.Surname = surname
	}

	if patronymic != "" {
		person.Patronymic = patronymic
	}

	if gender != "" {
		person.Gender = gender

	}

	if age != "" {
		// Преобразуем возраст в число
		ageInt, err := strconv.Atoi(age)
		if err != nil {
			return err
		}
		person.Age = ageInt
	}

	if nationality != "" {
		person.Nationality = nationality
	}

	// Сохраняем изменения в базе данных
	if err := app.DB.Save(&person).Error; err != nil {
		return err
	}

	// Возвращаем успешный ответ
	return c.SendString("Данные успешно изменены")
}

func (app *App) GetData(c *fiber.Ctx) error {
	// Получаем значения из запроса
	name := c.FormValue("name")
	surname := c.FormValue("surname")
	patronymic := c.FormValue("patronymic")
	gender := c.FormValue("gender")
	age := c.FormValue("age")
	nationality := c.FormValue("nationality")

	limit := c.FormValue("limit")
	page := c.FormValue("page")

	if limit == "" {
		limit = "10"
	}

	if page == "" {
		page = "1"
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		// Обработка ошибки преобразования limit
		return err
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		// Обработка ошибки преобразования page
		return err
	}

	// Вычисляем смещение (offset) на основе значения page и limit
	offset := (pageInt - 1) * limitInt

	var results []Person

	query := app.DB
	query = query.Offset(offset).Limit(limitInt)

	if name != "" {
		query = query.Where("name = ?", name)
	}
	if patronymic != "" {
		query = query.Where("patronymic =?", patronymic)
	}

	if gender != "" {
		query = query.Where("gender =?", gender)
	}

	if nationality != "" {
		query = query.Where("nationality =?", nationality)
	}

	if surname != "" {
		query = query.Where("surname =?", surname)
	}

	if age != "" {
		ageInt, err := strconv.Atoi(age)
		if err != nil {
			return err
		}
		query = query.Where("age = ?", ageInt)
	}

	if err := query.Find(&results).Error; err != nil {
		return err
	}

	return c.JSON(results)
}

func main() {
	app := &App{}

	time.Sleep(2 * time.Second)
	app.Initialize()

	appFiber := fiber.New()

	//в Процессе

	// Сделано
	appFiber.Get("/database", app.GetData)
	appFiber.Post("/database/edit", app.UpdateData)
	appFiber.Get("/", app.MainPage)
	appFiber.Delete("/database", app.DeleteData)
	appFiber.Post("/database", app.AddPerson)

	log.Fatal(appFiber.Listen(":3000"))
}

//// TODO: |||
//// env   |||
//// logs  |||
//// docs  |||
//// comms |||
