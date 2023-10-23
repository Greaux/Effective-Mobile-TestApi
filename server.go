package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func main() {
	app := &App{}

	time.Sleep(2 * time.Second)
	app.Initialize()

	appFiber := fiber.New()

	// appFiber.Get("/database/:Name", app.GetData)
	appFiber.Get("/", app.MainPage)
	// appFiber.Delete("/database/:uID", app.DeleteData)
	appFiber.Post("/database", app.AddPerson)

	log.Fatal(appFiber.Listen(":3000"))
}
