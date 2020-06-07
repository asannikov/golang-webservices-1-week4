package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	_ "fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type TestCase struct {
	Token   string
	Query   *SearchRequest
	Result  *SearchResponse
	IsError bool
}

type UserXml struct {
	ID        int    `xml:"id"`
	GUID      string `xml:"guid"`
	Active    bool   `xml:"isActive"`
	Balance   string `xml:"balance"`
	Picture   string `xml:"picture"`
	Age       int    `xml:"age"`
	EyeColor  string `xml:"eyeColor"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Gender    string `xml:"gender"`
	Company   string `xml:"company"`
	Email     string `xml:"email"`
	Phone     string `xml:"phone"`
	Address   string `xml:"address"`
	About     string `xml:"about"`
}

// Name sort
type UserNameSort []UserXml

func (slice UserNameSort) Len() int {
	return len(slice)
}

func (slice UserNameSort) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (slice UserNameSort) Less(i, j int) bool {
	return slice[i].FirstName+" "+slice[i].LastName < slice[j].FirstName+" "+slice[j].LastName
}

// Age sort
type UserAgeSort []UserXml

func (slice UserAgeSort) Len() int {
	return len(slice)
}

func (slice UserAgeSort) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (slice UserAgeSort) Less(i, j int) bool {
	return slice[i].Age < slice[j].Age
}

// Id sort
type UserIdSOrt []UserXml

func (slice UserIdSOrt) Len() int {
	return len(slice)
}

func (slice UserIdSOrt) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (slice UserIdSOrt) Less(i, j int) bool {
	return slice[i].ID < slice[j].ID
}

type Users struct {
	List []UserXml `xml:"row"`
}

func (usr *Users) FindUsers(query string, orderField string, limit int, offset int, soryby int) []User {

	if soryby != 0 {
		if orderField == "name" {
			result := UserNameSort(usr.List)
			if soryby == 1 {
				sort.Sort(sort.Reverse(result))
			} else {
				sort.Sort(result)
			}
		} else if orderField == "age" {
			result := UserAgeSort(usr.List)
			if soryby == 1 {
				sort.Sort(sort.Reverse(result))
			} else {
				sort.Sort(result)
			}
		} else if orderField == "id" {
			result := UserIdSOrt(usr.List)
			if soryby == 1 {
				sort.Sort(sort.Reverse(result))
			} else {
				sort.Sort(result)
			}
		}
	}

	var userFilter []User

	for _, userEnt := range usr.List {
		if query != "" && !strings.Contains(userEnt.FirstName+" "+userEnt.LastName, query) && !strings.Contains(userEnt.About, query) {
			continue
		}

		userFilter = append(userFilter, User{
			Id:     userEnt.ID,
			Name:   userEnt.FirstName + " " + userEnt.LastName,
			Age:    userEnt.Age,
			About:  userEnt.About,
			Gender: userEnt.Gender,
		})
	}

	var userList []User

	k := 0
	for i, userEnt := range userFilter {
		if k == limit {
			break
		}

		if i >= offset {
			userList = append(userList, User{
				Id:     userEnt.Id,
				Name:   userEnt.Name,
				Age:    userEnt.Age,
				About:  userEnt.About,
				Gender: userEnt.Gender,
			})

			k++
		}
	}

	return userList
}

func loadUsers() (Users, error) {
	xmlData, err := ioutil.ReadFile("./dataset.xml")
	if err != nil {
		panic(err)
	}
	v := new(Users)
	err = xml.Unmarshal(xmlData, &v)
	if err != nil {
		return Users{[]UserXml{}}, err
	}
	return *v, nil
}

func contains(arr [3]string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func handleRequest(r *http.Request) ([]User, error) {
	users, err := loadUsers()

	var userList []User

	if err != nil {
		return userList, err
	}

	fields := [3]string{}
	fields[0] = "name"
	fields[1] = "id"
	fields[2] = "age"

	var orderField string
	if contains(fields, strings.ToLower(r.FormValue("order_field"))) {
		orderField = strings.ToLower(r.FormValue("order_field"))
	} else if r.FormValue("order_field") == "" {
		orderField = "name"
	} else {
		return userList, errors.New("wrong_order_field_paramter")
	}

	limit := 10
	if r.FormValue("limit") != "" {
		limit, err = strconv.Atoi(r.FormValue("limit"))

		if err != nil {
			return userList, err
		}
	}

	offset := 0
	if r.FormValue("offset") != "" {
		offset, err = strconv.Atoi(r.FormValue("offset"))

		if err != nil {
			return userList, err
		}
	}

	sortby := 0
	if r.FormValue("order_by") != "" {
		sortby, err = strconv.Atoi(r.FormValue("order_by"))

		if err != nil {
			return userList, err
		}
	}

	result := users.FindUsers(r.FormValue("query"), orderField, limit, offset, sortby)

	return result, nil
}

/** эмулирует входящий запрос на сервер, который должен отдать соответствующий ответ*/
func QueryDummy(w http.ResponseWriter, r *http.Request) {

	StatusCode := 200
	var result []User
	var data []byte
	var err error

	if r.Header.Get("AccessToken") != "1234567890" {
		StatusCode = http.StatusUnauthorized
	} else {
		result, err = handleRequest(r)

		if err == nil {
			data, _ = json.Marshal(result)
		} else {
			StatusCode = http.StatusBadRequest
		}
	}

	//fmt.Println(result)
	//fmt.Println(data)
	switch StatusCode {
	case http.StatusBadRequest:
		if err.Error() == "wrong_order_field_paramter" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"ErrorBadOrderField"}`))
		} else {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"status": 400, "err": "user_read_error"}`)
		}
	case http.StatusUnauthorized:
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"status": 401, "err": "bad_access_tocken"}`)
	case 200:
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func TestSearchServer(t *testing.T) {
	cases := []TestCase{
		tResultCase1(),
		tResultCase2(),
		tResultCase3(),
		tResultCase4(),
		tResultCase5(),
		tResultCase6(),
		tResultCase7(),
		tResultCase8(),
		tResultCase9(),
		tResultCase10(),
		tResultCase11(),
		tResultCase12(),
		tResultCase13(),
	}

	ts := httptest.NewServer(http.HandlerFunc(QueryDummy))

	for caseNum, item := range cases {
		s := &SearchClient{
			AccessToken: item.Token,
			URL:         ts.URL,
		}
		result, err := s.FindUsers(*item.Query)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n, got \n %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestSearchServerInternalError(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Token: "1234567890",
			Query: &SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "Error",
				OrderField: "Name",
				OrderBy:    OrderByDesc,
			},
			Result:  nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	for caseNum, item := range cases {
		s := &SearchClient{
			AccessToken: item.Token,
			URL:         ts.URL,
		}
		result, err := s.FindUsers(*item.Query)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n, got \n %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestSearchServerTimeout(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Token: "1234567890",
			Query: &SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "Error",
				OrderField: "Name",
				OrderBy:    OrderByDesc,
			},
			Result:  nil,
			IsError: true,
		},
		TestCase{
			Token:   "1234567890",
			Query:   &SearchRequest{},
			Result:  nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))

	for caseNum, item := range cases {
		s := &SearchClient{
			AccessToken: item.Token,
			URL:         ts.URL,
		}
		result, err := s.FindUsers(*item.Query)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n, got \n %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestSearchServerBadRequestError(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Token: "1234567890",
			Query: &SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "Nulla cillum enim",
				OrderField: "Name",
				OrderBy:    OrderByDesc,
			},
			Result:  nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"unknown"}`))
	}))

	for caseNum, item := range cases {
		s := &SearchClient{
			AccessToken: item.Token,
			URL:         ts.URL,
		}
		result, err := s.FindUsers(*item.Query)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n, got \n %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestSearchServerUnknownError(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Token:   "1234567890",
			Query:   &SearchRequest{},
			Result:  nil,
			IsError: true,
		},
	}

	for caseNum, item := range cases {
		s := &SearchClient{}
		result, err := s.FindUsers(*item.Query)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n, got \n %#v", caseNum, item.Result, result)
		}
	}
}

func TestSearchServerJsonError(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Token:   "1234567890",
			Query:   &SearchRequest{},
			Result:  nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("notajson"))
	}))

	for caseNum, item := range cases {
		req := SearchRequest{}
		client := &SearchClient{
			AccessToken: item.Token,
			URL:         ts.URL,
		}

		result, err := client.FindUsers(req)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n, got \n %#v", caseNum, item.Result, result)
		}

	}

	ts.Close()
}

func TestSearchServerJsonError1(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Token:   "1234567890",
			Query:   &SearchRequest{},
			Result:  nil,
			IsError: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("notajson"))
	}))

	for caseNum, item := range cases {
		req := SearchRequest{}
		client := &SearchClient{
			AccessToken: item.Token,
			URL:         ts.URL,
		}

		result, err := client.FindUsers(req)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v\n, got \n %#v", caseNum, item.Result, result)
		}

	}

	ts.Close()
}

// =========================================================================================================

func tResultCase1() TestCase {
	result := []User{}
	result = append(result, User{
		Id:     0,
		Name:   "Boyd Wolf",
		Age:    22,
		About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
		Gender: "male",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      10,
			Offset:     0,
			Query:      "Nulla cillum enim",
			OrderField: "Name",
			OrderBy:    OrderByDesc,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: false,
		},
		IsError: false,
	}
}

func tResultCase2() TestCase {
	result := []User{}
	result = append(result, User{
		Id:     2,
		Name:   "Brooks Aguilar",
		Age:    25,
		About:  "Velit ullamco est aliqua voluptate nisi do. Voluptate magna anim qui cillum aliqua sint veniam reprehenderit consectetur enim. Laborum dolore ut eiusmod ipsum ad anim est do tempor culpa ad do tempor. Nulla id aliqua dolore dolore adipisicing.\n",
		Gender: "male",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      10,
			Offset:     0,
			Query:      "Aguilar",
			OrderField: "Name",
			OrderBy:    OrderByDesc,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: false,
		},
		IsError: false,
	}
}

func tResultCase3() TestCase {
	return TestCase{
		Token: "iasdasdas",
		Query: &SearchRequest{
			Limit:      10,
			Offset:     0,
			Query:      "Aguilar",
			OrderField: "Name",
			OrderBy:    OrderByDesc,
		},
		Result:  nil,
		IsError: true,
	}
}

func tResultCase4() TestCase {
	return TestCase{
		Token: "iasdasdas",
		Query: &SearchRequest{
			Limit:      -1,
			Offset:     0,
			Query:      "Aguilar",
			OrderField: "Name",
			OrderBy:    OrderByDesc,
		},
		Result:  nil,
		IsError: true,
	}
}

func tResultCase5() TestCase {
	return TestCase{
		Token: "iasdasdas",
		Query: &SearchRequest{
			Limit:      10,
			Offset:     -1,
			Query:      "Aguilar",
			OrderField: "Name",
			OrderBy:    OrderByDesc,
		},
		Result:  nil,
		IsError: true,
	}
}

func tResultCase6() TestCase {
	result := []User{}

	result = append(result, User{
		Id:     0,
		Name:   "Boyd Wolf",
		Age:    22,
		About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
		Gender: "male",
	})
	result = append(result, User{
		Id:     1,
		Name:   "Hilda Mayer",
		Age:    21,
		About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
		Gender: "female",
	})
	result = append(result, User{
		Id:     2,
		Name:   "Brooks Aguilar",
		Age:    25,
		About:  "Velit ullamco est aliqua voluptate nisi do. Voluptate magna anim qui cillum aliqua sint veniam reprehenderit consectetur enim. Laborum dolore ut eiusmod ipsum ad anim est do tempor culpa ad do tempor. Nulla id aliqua dolore dolore adipisicing.\n",
		Gender: "male",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      3,
			Offset:     0,
			Query:      "",
			OrderField: "Id",
			OrderBy:    OrderByAsIs,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: true,
		},
		IsError: false,
	}
}

func tResultCase7() TestCase {
	result := []User{}

	result = append(result, User{
		Id:     0,
		Name:   "Boyd Wolf",
		Age:    22,
		About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
		Gender: "male",
	})
	result = append(result, User{
		Id:     1,
		Name:   "Hilda Mayer",
		Age:    21,
		About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
		Gender: "female",
	})
	result = append(result, User{
		Id:     2,
		Name:   "Brooks Aguilar",
		Age:    25,
		About:  "Velit ullamco est aliqua voluptate nisi do. Voluptate magna anim qui cillum aliqua sint veniam reprehenderit consectetur enim. Laborum dolore ut eiusmod ipsum ad anim est do tempor culpa ad do tempor. Nulla id aliqua dolore dolore adipisicing.\n",
		Gender: "male",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      3,
			Offset:     0,
			Query:      "",
			OrderField: "Id",
			OrderBy:    OrderByAsc,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: true,
		},
		IsError: false,
	}
}

func tResultCase8() TestCase {
	result := []User{}

	result = append(result, User{
		Id:     34,
		Name:   "Kane Sharp",
		Age:    34,
		About:  "Lorem proident sint minim anim commodo cillum. Eiusmod velit culpa commodo anim consectetur consectetur sint sint labore. Mollit consequat consectetur magna nulla veniam commodo eu ut et. Ut adipisicing qui ex consectetur officia sint ut fugiat ex velit cupidatat fugiat nisi non. Dolor minim mollit aliquip veniam nostrud. Magna eu aliqua Lorem aliquip.\n",
		Gender: "male",
	})
	result = append(result, User{
		Id:     33,
		Name:   "Twila Snow",
		Age:    36,
		About:  "Sint non sunt adipisicing sit laborum cillum magna nisi exercitation. Dolore officia esse dolore officia ea adipisicing amet ea nostrud elit cupidatat laboris. Proident culpa ullamco aute incididunt aute. Laboris et nulla incididunt consequat pariatur enim dolor incididunt adipisicing enim fugiat tempor ullamco. Amet est ullamco officia consectetur cupidatat non sunt laborum nisi in ex. Quis labore quis ipsum est nisi ex officia reprehenderit ad adipisicing fugiat. Labore fugiat ea dolore exercitation sint duis aliqua.\n",
		Gender: "female",
	})
	result = append(result, User{
		Id:     32,
		Name:   "Christy Knapp",
		Age:    40,
		About:  "Incididunt culpa dolore laborum cupidatat consequat. Aliquip cupidatat pariatur sit consectetur laboris labore anim labore. Est sint ut ipsum dolor ipsum nisi tempor in tempor aliqua. Aliquip labore cillum est consequat anim officia non reprehenderit ex duis elit. Amet aliqua eu ad velit incididunt ad ut magna. Culpa dolore qui anim consequat commodo aute.\n",
		Gender: "female",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      3,
			Offset:     0,
			Query:      "",
			OrderField: "Id",
			OrderBy:    OrderByDesc,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: true,
		},
		IsError: false,
	}
}

func tResultCase9() TestCase {
	result := []User{}

	result = append(result, User{
		Id:     15,
		Name:   "Allison Valdez",
		Age:    21,
		About:  "Labore excepteur voluptate velit occaecat est nisi minim. Laborum ea et irure nostrud enim sit incididunt reprehenderit id est nostrud eu. Ullamco sint nisi voluptate cillum nostrud aliquip et minim. Enim duis esse do aute qui officia ipsum ut occaecat deserunt. Pariatur pariatur nisi do ad dolore reprehenderit et et enim esse dolor qui. Excepteur ullamco adipisicing qui adipisicing tempor minim aliquip.\n",
		Gender: "male",
	})
	result = append(result, User{
		Id:     16,
		Name:   "Annie Osborn",
		Age:    35,
		About:  "Consequat fugiat veniam commodo nisi nostrud culpa pariatur. Aliquip velit adipisicing dolor et nostrud. Eu nostrud officia velit eiusmod ullamco duis eiusmod ad non do quis.\n",
		Gender: "female",
	})
	result = append(result, User{
		Id:     19,
		Name:   "Bell Bauer",
		Age:    26,
		About:  "Nulla voluptate nostrud nostrud do ut tempor et quis non aliqua cillum in duis. Sit ipsum sit ut non proident exercitation. Quis consequat laboris deserunt adipisicing eiusmod non cillum magna.\n",
		Gender: "male",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      3,
			Offset:     0,
			Query:      "",
			OrderField: "Name",
			OrderBy:    OrderByAsc,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: true,
		},
		IsError: false,
	}
}

func tResultCase10() TestCase {
	result := []User{}

	result = append(result, User{
		Id:     13,
		Name:   "Whitley Davidson",
		Age:    40,
		About:  "Consectetur dolore anim veniam aliqua deserunt officia eu. Et ullamco commodo ad officia duis ex incididunt proident consequat nostrud proident quis tempor. Sunt magna ad excepteur eu sint aliqua eiusmod deserunt proident. Do labore est dolore voluptate ullamco est dolore excepteur magna duis quis. Quis laborum deserunt ipsum velit occaecat est laborum enim aute. Officia dolore sit voluptate quis mollit veniam. Laborum nisi ullamco nisi sit nulla cillum et id nisi.\n",
		Gender: "male",
	})
	result = append(result, User{
		Id:     33,
		Name:   "Twila Snow",
		Age:    36,
		About:  "Sint non sunt adipisicing sit laborum cillum magna nisi exercitation. Dolore officia esse dolore officia ea adipisicing amet ea nostrud elit cupidatat laboris. Proident culpa ullamco aute incididunt aute. Laboris et nulla incididunt consequat pariatur enim dolor incididunt adipisicing enim fugiat tempor ullamco. Amet est ullamco officia consectetur cupidatat non sunt laborum nisi in ex. Quis labore quis ipsum est nisi ex officia reprehenderit ad adipisicing fugiat. Labore fugiat ea dolore exercitation sint duis aliqua.\n",
		Gender: "female",
	})
	result = append(result, User{
		Id:     18,
		Name:   "Terrell Hall",
		Age:    27,
		About:  "Ut nostrud est est elit incididunt consequat sunt ut aliqua sunt sunt. Quis consectetur amet occaecat nostrud duis. Fugiat in irure consequat laborum ipsum tempor non deserunt laboris id ullamco cupidatat sit. Officia cupidatat aliqua veniam et ipsum labore eu do aliquip elit cillum. Labore culpa exercitation sint sint.\n",
		Gender: "male",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      3,
			Offset:     0,
			Query:      "",
			OrderField: "Name",
			OrderBy:    OrderByDesc,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: true,
		},
		IsError: false,
	}
}

func tResultCase11() TestCase {
	return TestCase{
		Token: "iasdasdas",
		Query: &SearchRequest{
			Limit:      10,
			Offset:     0,
			Query:      "Aguilar",
			OrderField: "picture",
			OrderBy:    OrderByDesc,
		},
		Result:  nil,
		IsError: true,
	}
}

func tResultCase12() TestCase {
	result := []User{}

	result = append(result, User{
		Id:     33,
		Name:   "Twila Snow",
		Age:    36,
		About:  "Sint non sunt adipisicing sit laborum cillum magna nisi exercitation. Dolore officia esse dolore officia ea adipisicing amet ea nostrud elit cupidatat laboris. Proident culpa ullamco aute incididunt aute. Laboris et nulla incididunt consequat pariatur enim dolor incididunt adipisicing enim fugiat tempor ullamco. Amet est ullamco officia consectetur cupidatat non sunt laborum nisi in ex. Quis labore quis ipsum est nisi ex officia reprehenderit ad adipisicing fugiat. Labore fugiat ea dolore exercitation sint duis aliqua.\n",
		Gender: "female",
	})

	result = append(result, User{
		Id:     34,
		Name:   "Kane Sharp",
		Age:    34,
		About:  "Lorem proident sint minim anim commodo cillum. Eiusmod velit culpa commodo anim consectetur consectetur sint sint labore. Mollit consequat consectetur magna nulla veniam commodo eu ut et. Ut adipisicing qui ex consectetur officia sint ut fugiat ex velit cupidatat fugiat nisi non. Dolor minim mollit aliquip veniam nostrud. Magna eu aliqua Lorem aliquip.\n",
		Gender: "male",
	})

	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      40,
			Offset:     33,
			Query:      "",
			OrderField: "Id",
			OrderBy:    OrderByAsc,
		},
		Result: &SearchResponse{
			Users:    result,
			NextPage: false,
		},
		IsError: false,
	}
}

func tResultCase13() TestCase {
	return TestCase{
		Token: "1234567890",
		Query: &SearchRequest{
			Limit:      40,
			Offset:     33,
			Query:      "",
			OrderField: "WrongField",
			OrderBy:    OrderByAsc,
		},
		Result:  nil,
		IsError: true,
	}
}
