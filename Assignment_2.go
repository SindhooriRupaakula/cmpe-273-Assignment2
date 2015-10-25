package main

import (
"fmt"
"strings"
"net/http"
"io/ioutil"
"encoding/json"
"gopkg.in/mgo.v2"
"gopkg.in/mgo.v2/bson"
"github.com/julienschmidt/httprouter"
)



type GoogleCoordinates struct {
	Results []struct {
	
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry struct {
			
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			
			LocationType string `json:"location_type"`
			
			Viewport     struct {
				
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PlaceID string   `json:"place_id"`
		Types   []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

type Response struct {

      Id bson.ObjectId `json:"id" bson:"_id"`
      Name string `json:"name" bson:"name"`
      Address string `json:"address" bson:"address"`
      City string `json:"city" bson:"city"`
      State string `json:"state" bson:"state"`
      Zip string `json:"zip" bson:"zip"`
      Coordinate struct 
	  {
	   Lat float64 `json:"lat"   bson:"lat"`
	   Lng float64 `json:"lng"   bson:"lng"`		
	  }`json:"coordinate" bson:"coordinate"`
}
		

type MongoSession struct {
				session *mgo.Session
			}

			
func newMongoSession(session *mgo.Session) *MongoSession {
	return &MongoSession{session}
}

func (ms MongoSession) GetLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	
	id := params.ByName("id")
    if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	
	fmt.Print("Before OID")
	oid := bson.ObjectIdHex(id)
	fmt.Print("OID is", oid)

	resp := Response{}
	
	if err := ms.session.DB("cmpe273").C("locations").FindId(oid).One(&resp); err != nil {
		fmt.Print("Inside fail case")
		w.WriteHeader(404)
		return
	}

	json.NewDecoder(r.Body).Decode(resp)

	mObject, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", mObject)
}



func (ms MongoSession) CreateLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {


	resp := Response{}

	json.NewDecoder(r.Body).Decode(&resp)

	data := callGoogleAPI(&resp)
	
	data.Id = bson.NewObjectId()

	ms.session.DB("cmpe273").C("locations").Insert(data)
	
	mObject, _ := json.Marshal(data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", mObject)
}


func (ms MongoSession) DeleteLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	    
	    id := params.ByName("id")

	   
	    if !bson.IsObjectIdHex(id) {
	        w.WriteHeader(404)
	        return
	    }

	    oid := bson.ObjectIdHex(id)
	    if err := ms.session.DB("cmpe273").C("locations").RemoveId(oid); err != nil {
		    fmt.Print("Inside fail case")
	        w.WriteHeader(404)
	        return
	    }
	   
	    w.WriteHeader(200)
}


func (ms MongoSession) UpdateLocation (w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	
	id := params.ByName("id")

	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	oid := bson.ObjectIdHex(id)

	get := Response{}
	put := Response{}

	put.Id = oid

	json.NewDecoder(r.Body).Decode(&put)

	if err := ms.session.DB("cmpe273").C("locations").FindId(oid).One(&get); err != nil {
		w.WriteHeader(404)
		return
	}

	na := get.Name

	object := ms.session.DB("cmpe273").C("locations")

	get = callGoogleAPI(&put)
	object.Update(bson.M{"_id": oid}, bson.M{"$set": bson.M{ "address": put.Address, "city": put.City, "state": put.State, "zip" : put.Zip, "coordinate": bson.M{"lat" : get.Coordinate.Lat, "lng" : get.Coordinate.Lng}}})

	get.Name = na

	mObject, _ := json.Marshal(get)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", mObject)

}

func callGoogleAPI (resp *Response) Response {

  address := resp.Address
  city := resp.City

  gstate := strings.Replace(resp.State," ","+",-1)
  gaddress := strings.Replace(address, " ", "+", -1)
  gcity := strings.Replace(city," ","+",-1)

	uri := "http://maps.google.com/maps/api/geocode/json?address="+gaddress+"+"+gcity+"+"+gstate+"&sensor=false"


    result, _ := http.Get(uri)

	body, _ := ioutil.ReadAll(result.Body)


 	Cords := GoogleCoordinates{}

    err := json.Unmarshal(body, &Cords)
    if err!= nil {
      panic(err)
    } 


	 for _, Sample := range Cords.Results {
				resp.Coordinate.Lat= Sample.Geometry.Location.Lat
				resp.Coordinate.Lng = Sample.Geometry.Location.Lng
		}

   return *resp
}


func getConnection() *mgo.Session {

    conn, err := mgo.Dial("mongodb://sindh:sindh123@ds045064.mongolab.com:45064/assig1")

    if err != nil {
        panic(err)
    }
    return conn
}

func main() {

    r := httprouter.New()
 
  	ms := newMongoSession(getConnection())
	
  	r.GET("/locations/:id", ms.GetLocation)
  	r.POST("/locations",ms.CreateLocation)
	r.DELETE("/locations/:id",ms.DeleteLocation)
	r.PUT("/locations/:id", ms.UpdateLocation)
	
	http.ListenAndServe("localhost:8080",r)

}
