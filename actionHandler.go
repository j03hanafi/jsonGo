package main

import (
	"encoding/json"
	"fmt"
	"github.com/mofax/iso8583"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
}

// handler action from route with request get all payments
// todo
// return error from query
// get limit
func getPayments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := PaymentsResponse{}
	err := pingDb(dbCon)

	if err != nil {
		w.WriteHeader(500)
		response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 500, serverError
	} else {
		payments, err := selectPayments(dbCon)
		if err != nil {
			w.WriteHeader(500)
			response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 500, err.Error()
		} else {
			w.WriteHeader(200)
			response.ResponseStatus.ResponseCode, response.ResponseStatus.ResponseDescription = 200, "success"
			response.TransactionData = payments
		}
	}

	json.NewEncoder(w).Encode(response)
}

func writeFile(transaction Transaction) {

	one := iso8583.NewISOStruct("spec1987.yml", false)

	if one.Mti.String() != "" {
		fmt.Printf("Empty generates invalid MTI")
	}

	one.AddMTI("0200")
	one.AddField(2, transaction.Pan)
	one.AddField(3, transaction.ProcessingCode)
	one.AddField(4, strconv.Itoa(transaction.TotalAmount))
	one.AddField(5, transaction.SettlementAmount)
	one.AddField(6, transaction.CardholderBillingAmount)
	one.AddField(7, transaction.TransmissionDateTime)
	one.AddField(9, transaction.SettlementConversionrate)
	one.AddField(10, transaction.CardHolderBillingConvRate)
	one.AddField(11, transaction.Stan)
	one.AddField(12, transaction.LocalTransactionTime)
	one.AddField(13, transaction.LocalTransactionDate)
	one.AddField(17, transaction.CaptureDate)
	one.AddField(18, transaction.CategoryCode)
	one.AddField(22, transaction.PointOfServiceEntryMode)
	one.AddField(37, transaction.Refnum)
	one.AddField(41, transaction.CardAcceptorData.CardAcceptorTerminalId)
	one.AddField(43, transaction.CardAcceptorData.CardAcceptorName)
	one.AddField(48, transaction.AdditionalData)
	one.AddField(49, transaction.Currency)
	one.AddField(50, transaction.SettlementCurrencyCode)
	one.AddField(51, transaction.CardHolderBillingCurrencyCode)
	one.AddField(57, transaction.AdditionalDataNational)

	expected := "02007ef8c40008a1e080199360014900000008883263010115500001350000135000009210820220000011100000111554461082022092109217011011678615554461C01IUT MLPT      RINTIS     050PI04Q001CD30SUSAEN                         MC03UMI36070270202061051511562070703C01"
	unpacked, _ := one.ToString()
	if unpacked != expected {
		fmt.Printf("Manually constructed isostruct produced %s not %s", unpacked, expected)
	}

	//Check if file's name already exist
	fileName := transaction.ProcessingCode
	if !strings.Contains(fileName, ".txt") {
		fileName += ".txt"
	}

	content := CreateFile(fileName, unpacked)

	fmt.Printf(content)
}

func CreateFile(fileName string, content string) string {

	if !strings.Contains(fileName, ".txt") {
		fileName += ".txt"
	}

	file, err := os.Create("files/" + fileName)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	defer file.Close()

	_, err = file.WriteString(content)

	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}

	return content

}

// handler action from route with request get payment with
// procid
// todo
// return error from query
func getPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := PaymentResponse{}
	err := pingDb(dbCon)
	processingCode := mux.Vars(r)["id"]

	if err != nil {
		w.WriteHeader(500)
		response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 500, serverError
	} else {
		payment, err := selectPayment(processingCode, dbCon)
		if err != nil {
			w.WriteHeader(500)
			response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 500, err.Error()
		} else if payment.ProcessingCode == "" {
			w.WriteHeader(404)
			response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 404, "data not found"

		} else {
			w.WriteHeader(200)
			response.ResponseStatus.ResponseCode, response.ResponseStatus.ResponseDescription = 200, "success"
			response.TransactionData = payment
			writeFile(payment)
		}
	}
	json.NewEncoder(w).Encode(response)
}

//handler action from route with request post with json body required
//todo
//check if json in correct format
//return error from query
func createPayment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	b, err := ioutil.ReadAll(r.Body)
	errorCheck(err)
	response := PaymentResponse{}

	err = pingDb(dbCon)

	if err != nil {
		response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 500, serverError
	} else {
		var trs Transaction
		err = json.Unmarshal(b, &trs)

		errorCheck(err)

		if checkExistence(trs.ProcessingCode, dbCon) {
			processingCode, err := insertPayment(trs, dbCon)

			if err != nil {
				w.WriteHeader(500)
				response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 500, err.Error()
			} else {
				w.WriteHeader(200)
				response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 200, "success"
				response.TransactionData, _ = selectPayment(processingCode, dbCon)
			}
		} else {
			w.WriteHeader(403)
			response.ResponseStatus.ResponseCode, response.ResponseStatus.ResponseDescription = 403, "duplicate processingCode"
			response.TransactionData = trs
		}
	}

	json.NewEncoder(w).Encode(response)
}

//handler action from route with request put with json body required
//todo
//check if json in correct format
//return error from query
func updatePayment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)

	w.Header().Set("Content-Type", "application/json")
	errorCheck(err)

	var trs Transaction
	var canQue []string
	var as string
	err = json.Unmarshal(b, &trs)

	s := reflect.ValueOf(&trs).Elem()
	t := reflect.ValueOf(&trs.CardAcceptorData).Elem()
	typeOfT := s.Type()
	typeOfU := t.Type()
	procCode := mux.Vars(r)["id"]
	response := PaymentResponse{}

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)

		typeOfF := fmt.Sprintf(typeOfT.Field(i).Type.Name())

		if typeOfF == "CardAcceptorData" {
			for j := 0; j < t.NumField(); j++ {
				g := t.Field(j)
				val := g.Interface()
				if val != "" && val != "0" {
					as = fmt.Sprintf("%s='%v'", typeOfU.Field(j).Name, g.Interface())
					canQue = append(canQue, as)
				}
			}
		} else {
			val := f.Interface()
			if val != "" && val != 0 {
				as = fmt.Sprintf("%s='%v'", typeOfT.Field(i).Name, f.Interface())
				canQue = append(canQue, as)
			}
		}
	}

	preQue := strings.Join(canQue, ", ")
	exeQue := fmt.Sprintln("UPDATE transaction SET " + preQue + " where processingCode =" + procCode)
	payment, _ := selectPayment(procCode, dbCon)
	err = putPayment(exeQue, dbCon)

	if err != nil {
		w.WriteHeader(500)
		response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 500, err.Error()
	} else {
		if payment.ProcessingCode == "" {
			w.WriteHeader(400)
			response.ResponseStatus.ReasonCode, response.ResponseStatus.ResponseDescription = 400, "data not exist"
		} else {
			w.WriteHeader(200)
			response.ResponseStatus.ResponseCode, response.ResponseStatus.ResponseDescription = 200, "updated"
			response.TransactionData = trs
		}
	}

	json.NewEncoder(w).Encode(response)
}

//handler action from route with request delte based on proc code that send as param
//todo
//return error from query
func deletePayment(w http.ResponseWriter, r *http.Request) {
	response := DelPaymentResponse{}
	w.Header().Set("Content-Type", "application/json")
	err := pingDb(dbCon)

	procCode := mux.Vars(r)["id"]
	if err != nil {
		w.WriteHeader(500)
		response.ResponseCode, response.ResponseDescription = 500, serverError
	} else {
		payment, _ := selectPayment(procCode, dbCon)
		if payment.ProcessingCode == "" {
			w.WriteHeader(404)
			response.ResponseCode, response.ResponseDescription = 404, "data not exist"
		} else {
			dropPayment(procCode, dbCon)
			w.WriteHeader(200)
			response.ResponseCode, response.ResponseDescription = 200, "Deleted"
		}
	}

	json.NewEncoder(w).Encode(response)
}
