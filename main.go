package main

import (
	"encoding/json"
	"github.com/graphql-go/graphql"
	"github.com/mpenick/robot/control"
	"github.com/rs/cors"
	"log"
	"net/http"
)

var ctrl *control.Control

type requestBody struct {
	Query string `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

var moveType = graphql.NewEnum(graphql.EnumConfig{
	Name: "MoveType",
	Values: graphql.EnumValueConfigMap{
		"FORWARD":  &graphql.EnumValueConfig{Value: control.Forward},
		"BACKWARD": &graphql.EnumValueConfig{Value: control.Backward},
		"LEFT":     &graphql.EnumValueConfig{Value: control.Left},
		"RIGHT":    &graphql.EnumValueConfig{Value: control.Right},
	},
})

var queryType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Query",
	Fields: graphql.Fields{
		"moves": &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(moveType)),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return nil, nil
			},
		},
	},
})

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		"move": &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(moveType)),
			Args: graphql.FieldConfigArgument{
				"type": &graphql.ArgumentConfig{
					Type: moveType,
				},
				"user": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				//ctrl.Move(p.Args["type"].(control.Move))
				return make([]control.Move, 0), nil
			},
		},
	},
})

var schema, _ = graphql.NewSchema(
	graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})

func main() {
	var err error
	//ctrl, err = control.NewControl()
	if err != nil {
		log.Fatalf("unable to create robot control: %v", err)
	}

	cors := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		Debug:          true,
	})

	//_ = ctrl
	//<-time.After(5 * time.Second)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.Handle("/graphql", cors.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "no request body", 400)
			return
		}

		cookie := r.Header["Cookie"]
		_ = cookie

		var body requestBody
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			http.Error(w, "request body is invalid", 400)
			return
		}

		result := graphql.Do(graphql.Params{
			Schema:        schema,
			RequestString: body.Query,
			Context:       r.Context(),
			VariableValues: body.Variables,
		})
		json.NewEncoder(w).Encode(result)

	})))
	http.ListenAndServe(":8080", nil)
}
