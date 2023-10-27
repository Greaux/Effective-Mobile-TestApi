package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// общая структура для БД/записей
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

	// пробуем открыть .env который лежит локально
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	/*
		logmod := os.Getenv("LOGGING")

		if logmod == "DEBUG" {
		}
		if logmod == "INFO" {
		}
	*/

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSL_MODE"),
	)

	//Подключаемся к БД (можно расскоментить если хотит захардкодить вместо .env)
	//dsn := "host=db user=postgres password=XDDDPASSW0RD dbname=EFMOBILPersons port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	app.DB = db

	// Автоматическая миграция в БД
	app.DB.AutoMigrate(&Person{})
}

// Страница затычка (проверять запущен сервер или нет)
func (app *App) MainPage(c *fiber.Ctx) error {
	c.Status(http.StatusOK)
	return c.SendString("Тут ничего нет.")
}

// запрос к АПИшке для получения возраста
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

// запрос  к АПИшке для получения национальности (берём самое вероятное)
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

// Запрос получения пола по АПИшке
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

// Добавление человека в БД
func (app *App) AddPerson(c *fiber.Ctx) error {
	name := c.FormValue("name")
	surname := c.FormValue("surname")
	patronymic := c.FormValue("patronymic")

	if len(name) == 0 || len(surname) == 0 {
		c.Status(http.StatusBadRequest)
		return c.SendString("Необходимо указать фамилию и имя.")
	}

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

	// Записываем результат в БД
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

// Функция удаления записи человека из БД (по id)
func (app *App) DeleteData(c *fiber.Ctx) error {
	uid := c.FormValue("id")
	var person Person

	if uid == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID is required",
		})
	}

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

// функция изменения записи в БД
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

	// проверка перед запросом, чтоб не нагружать БД
	if name == "" && surname == "" && patronymic == "" && gender == "" && age == "" && nationality == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You must specify at least one parameter",
		})
	}

	// получаем запись с БД
	if err := app.DB.First(&person, id).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
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
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		person.Age = ageInt
	}

	if nationality != "" {
		person.Nationality = nationality
	}

	// Сохраняем изменения в базе данных
	if err := app.DB.Save(&person).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Возвращаем успешный ответ
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Данные успешно изменены",
	})
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

	// проверка перед запросом, чтоб не нагружать БД
	if name == "" && surname == "" && patronymic == "" && gender == "" && age == "" && nationality == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You must specify at least one parameter",
		})
	}

	if limit == "" {
		limit = "10"
	}

	if page == "" {
		page = "1"
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		// Обработка ошибки преобразования limit
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		// Обработка ошибки преобразования page
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
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
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		query = query.Where("age = ?", ageInt)
	}

	if err := query.Find(&results).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(results)
}

func main() {
	app := &App{}

	time.Sleep(2 * time.Second)
	app.Initialize()
	appFiber := fiber.New()

	//в Процессе
	appFiber.Get("/database", app.GetData)
	appFiber.Post("/database/edit", app.UpdateData)
	appFiber.Get("/", app.MainPage)
	appFiber.Delete("/database", app.DeleteData)
	appFiber.Post("/database", app.AddPerson)

	// Получение порта и ip из .env
	port := os.Getenv("APP_PORT")
	ip := os.Getenv("APP_ADDRESS")

	if port == "" {
		port = "3000"
	}

	log.Fatal(appFiber.Listen(ip + ":" + port))
}

//// TODO: |||
//// logs 50/50 |||
//// docs  |||
//// comms написать/переписать |||
