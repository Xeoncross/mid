package mid

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/schema"
	"github.com/julienschmidt/httprouter"
)

// ValidationErrors occurs whenever one or more fields fail the validation by govalidator
type ValidationErrors map[string]string

//
// func (v ValidationErrors) Error() string {
// 	s := []string{}
// 	for k, v := range v {
// 		s = append(s, fmt.Sprintf("%s: %s", k, v))
// 	}
// 	return fmt.Sprintf("Validation error: %s", strings.Join(s, ","))
// }

// ValidateStruct provided returning a ValidationErrors or error
func ValidateStruct(h reflect.Value, hc handlerContext, r *http.Request, ps httprouter.Params) (err error, validation ValidationErrors) {

	// if r.Header.Get("Content-Type") == "application/json" {
	if hc.body {

		body := h.FieldByName(FieldBody)
		b := body.Addr().Interface()

		err = json.NewDecoder(r.Body).Decode(b)
		// fmt.Printf("Decoded JSON: %+v\n", b)

		if err != nil {
			// We don't care about type errors
			// the validator will handle those messages better below
			switch err.(type) {
			// json: cannot unmarshal string into Go struct field A.Foo of type foo.Bar
			case *json.UnmarshalTypeError:
				// fmt.Printf("Decoded JSON: %+v\n", b)
				// err = fmt.Errorf("JSON: Unexpected type '%s' for field '%s'", e.Value, e.Field)
				// log.Printf("UnmarshalTypeError: Value[%s] Type[%v]\n", e.Value, e.Type)
			case *json.InvalidUnmarshalError:
				// log.Printf("InvalidUnmarshalError: Type[%v]\n", e.Type)
			// unexpected EOF
			default:
				// We could just ignore all JSON errors like we do with gorilla/schema
				// However, JSON errors should be rare and could make development
				// a lot harder if something weird happens. Better alert the client.
				// return fmt.Errorf("Invalid JSON: %s", err.Error()), validation
				return
			}
		}

	} else if hc.form { // GET or application/x-www-form-urlencoded

		form := h.FieldByName(FieldForm)
		f := form.Addr().Interface()

		// Parse the input (Already called if using DefaultHandlers)
		r.ParseForm()

		// 1. Try to insert form data into the struct
		decoder := schema.NewDecoder()

		// A) Developer forgot about a field
		// B) Client is messing with the request fields
		decoder.IgnoreUnknownKeys(true)

		// Edge Case: https://github.com/gorilla/schema/blob/master/decoder.go#L203
		// "schema: converter not found for..."

		// gorilla/schema errors share application handler structure which is
		// not safe for us, nor helpful to our clients
		decoder.Decode(f, r.Form)
	}

	// Query params?
	if hc.query {
		query := h.FieldByName(FieldQuery)
		queryType := query.Type()

		queryValues := r.URL.Query()
		for i := 0; i < query.NumField(); i++ {
			field := queryType.Field(i)

			var s string
			tag, ok := field.Tag.Lookup(TagQuery)
			if ok {
				s = queryValues.Get(tag)
			} else {
				s = queryValues.Get(field.Name)
			}

			f := query.Field(i)
			f.SetString(s)
		}
	}

	if hc.param {
		param := h.FieldByName(FieldParameter)
		paramType := param.Type()

		numFields := paramType.NumField()
		for i := 0; i < numFields; i++ {
			field := paramType.Field(i)

			s := ps.ByName(field.Name)
			val := param.Field(i)
			val.SetString(s)

		}
	}

	// 2. Validate the struct data rules
	var isValid bool
	isValid, err = govalidator.ValidateStruct(h.Interface())

	if !isValid {
		validation = ValidationErrors(govalidator.ErrorsByField(err))
	}

	return nil, validation
}
