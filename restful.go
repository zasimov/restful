package restful

import (
	"fmt"
	"log"
        "encoding/json"
        "net/http"
	"github.com/gorilla/mux"
	"github.com/nu7hatch/gouuid"
)


/* Special functions */
func correctCollectionPath(collectionPath string) (string) {
	if collectionPath == "" {
		return "/"
	}
	if collectionPath[len(collectionPath) - 1] != '/' {
		return collectionPath + "/"
	}
	return collectionPath
}


/* Basic HTTP codes */
const (
	ScOK = 200
	ScCreated = 201
	ScNoContent = 204
        ScInternalServerError = 500
        ScBadRequest = 400
	ScNotFound = 404
        ScMethodNotAllowed = 405
        ScConflict = 409
	ScUnprocessableEntity = 422
)


const UUID = "uuid"

func placeholder(name string) (string) {
	return "{" + name + "}"
}


const APPLICATION_JSON = "application/json"
const PLAIN_TEXT = "text/plain"


type Service struct {
	Router *mux.Router
	Context interface{}
}


type Request struct {
	RequestID string
	Service *Service
	HttpRequest *http.Request
}


type IController interface {
	RootUrl() (string)
        Create(request Request) (Response)
        Get(request Request) (Response)
	Update(request Request) (Response)
        Delete(request Request) (Response)
	List(request Request) (Response)
}


type Controller struct {
	Url string
}


func (controller Controller) RootUrl() string {
	return correctCollectionPath(controller.Url)
}

func (controller Controller) Location(uuid string) string {
	return controller.RootUrl() + uuid
}


/* UUID Helpers */
func genuuid4() (string, error) {
	uuid4, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return uuid4.String(), err
}


func initRequest(service *Service, httpRequest *http.Request) (request Request) {
	var err error
	request.RequestID, err = genuuid4()
	if err != nil {
		log.Fatal(err)
	}
	request.Service = service
	request.HttpRequest = httpRequest
	return request
}


type Response struct {
	Code int
	ContentType string
	Location string
	Uuid string
	Headers map[string]string
	Data []byte
}


func Created(uuid string, location string) (response Response) {
	response = Response{
		Code: ScCreated,
		Location: location,
		Uuid: uuid}
	return response
}

func Updated() (response Response) {
	response = Response{
		Code: ScCreated}
	return response
}

func Deleted() (response Response) {
	response = Response{
		Code: ScCreated}
	return response
}

func BadRequest() (response Response) {
	response = Response{
		Code: ScBadRequest}
	return response
}


func InternalServerError(errinfo string) (response Response) {
	response = Response{
		Code: ScInternalServerError,
		ContentType: PLAIN_TEXT,
		Data: []byte(errinfo)}
	return response
}


func NotFound() (response Response) {
	response = Response{
		Code: ScNotFound}
	return response
}


func Conflict() (response Response) {
	response = Response{
		Code: ScConflict}
	return response
}


func MethodNotAllowed() (response Response) {
	response = Response{
		Code: ScMethodNotAllowed,
		Data: nil}
	return response
}


func UnprocessableEntity(message string) (response Response) {
	response = Response{
		Code: ScUnprocessableEntity,
		Data: []byte(message)}
	return response
}



func Plain(answer string) (response Response) {
	response = Response{
		ContentType: PLAIN_TEXT,
		Code: ScOK,
		Data: []byte(answer),
	}
	return response
}


func Json(obj interface{}) (response Response) {
	jsonContent, err := json.Marshal(obj)
	if err != nil {
		return InternalServerError(err.Error())
	} else {
		response = Response{
			ContentType: APPLICATION_JSON,
			Code: ScOK,
			Data: jsonContent,
		}
		return response
	}
}


func (*Controller) Create(request Request) (Response) {
	return MethodNotAllowed()
}

func (*Controller) Get(request Request) (Response) {
	return MethodNotAllowed()
}

func (*Controller) Update(request Request) (Response) {
	return MethodNotAllowed()
}

func (*Controller) Delete(request Request) (Response) {
	return MethodNotAllowed()
}

func (*Controller) List(request Request) (Response) {
	return MethodNotAllowed()
}

func (request Request) JsonDecoder() *json.Decoder {
	return json.NewDecoder(request.HttpRequest.Body)
}

func (request Request) Vars() map[string]string {
	return mux.Vars(request.HttpRequest)
}

func (request Request) Var(name string) string {
	return request.Vars()[name]
}

func (request Request) ResourceUuid() string {
	return request.Var(UUID)
}

func NewService(context interface{}) (service *Service) {
	service = new(Service)
	service.Router = mux.NewRouter()
	service.Context = context
	http.Handle("/", service.Router)
	return service
}


func logRequest(request Request) {
	log.Println(request.RequestID,
  		    "Request",
                    request.HttpRequest.Method,
		    request.HttpRequest.URL)
}


func logResponse(request Request, response Response) {
	log.Println(request.RequestID,
  		    "Response",
  	            response.Code)
}


func SendResponse(request Request, rw http.ResponseWriter, response Response) () {
	logResponse(request, response)

	if response.ContentType != "" {
		rw.Header().Set("Content-Type", response.ContentType)
	}
	if response.Location != "" {
		rw.Header().Set("Location", response.Location)
	}
	if response.Uuid != "" {
		rw.Header().Set("X-UUID", response.Uuid)
	}

	rw.WriteHeader(response.Code)

	if len(response.Data) != 0 {
		rw.Write(response.Data)
	}
}


func (service *Service) constructItemHandler(controller IController) http.HandlerFunc {
        return func(rw http.ResponseWriter, httpRequest *http.Request) {

		request := initRequest(service, httpRequest)
		logRequest(request)

                method := httpRequest.Method

		var response Response

                switch method {
                case "GET":
                        response = controller.Get(request)
                case "PUT":
                        response = controller.Update(request)
                case "DELETE":
                        response = controller.Delete(request)
		default:
			response = MethodNotAllowed()
                }

		SendResponse(request, rw, response)
        }
}


func (service *Service) constructCollectionHandler(controller IController) http.HandlerFunc {
        return func(rw http.ResponseWriter, httpRequest *http.Request) {

		request := initRequest(service, httpRequest)
		logRequest(request)

                method := httpRequest.Method

		var response Response

                switch method {
                case "GET":
                        response = controller.List(request)
                case "POST":
                        response = controller.Create(request)
		default:
			response = MethodNotAllowed()
                }

		SendResponse(request, rw, response)
        }
}


func (service *Service) Register(controller IController) {
	collectionPath := controller.RootUrl()
	correctedPath := correctCollectionPath(collectionPath)
	service.Router.HandleFunc(
		correctedPath,
		service.constructCollectionHandler(controller))

	service.Router.HandleFunc(
		correctedPath + "{uuid}",
		service.constructItemHandler(controller))
}

func (service *Service) RegisterAction(controller IController) {
	collectionPath := controller.RootUrl()
	correctedPath := correctCollectionPath(collectionPath) + "invoke"
	service.Router.HandleFunc(
		correctedPath,
		service.constructCollectionHandler(controller))
}

func (service *Service) Forever(host string, port uint) {
	address := fmt.Sprintf("%s:%d", host, port)
	log.Println("Forever on", address)
	http.ListenAndServe(address, nil)
}
