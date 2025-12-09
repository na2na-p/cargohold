package usecase

import "encoding/json"

type BatchResponse struct {
	transfer string
	objects  []ResponseObject
	hashAlgo string
}

func NewBatchResponse(transfer string, objects []ResponseObject, hashAlgo string) BatchResponse {
	objectsCopy := make([]ResponseObject, len(objects))
	copy(objectsCopy, objects)
	return BatchResponse{
		transfer: transfer,
		objects:  objectsCopy,
		hashAlgo: hashAlgo,
	}
}

func (r BatchResponse) Transfer() string {
	return r.transfer
}

func (r BatchResponse) Objects() []ResponseObject {
	result := make([]ResponseObject, len(r.objects))
	copy(result, r.objects)
	return result
}

func (r BatchResponse) HashAlgo() string {
	return r.hashAlgo
}

type batchResponseJSON struct {
	Transfer string               `json:"transfer"`
	Objects  []responseObjectJSON `json:"objects"`
	HashAlgo string               `json:"hash_algo,omitempty"`
}

func (r BatchResponse) MarshalJSON() ([]byte, error) {
	objects := make([]responseObjectJSON, len(r.objects))
	for i, obj := range r.objects {
		objects[i] = obj.toJSON()
	}
	return json.Marshal(batchResponseJSON{
		Transfer: r.transfer,
		Objects:  objects,
		HashAlgo: r.hashAlgo,
	})
}

func (r *BatchResponse) UnmarshalJSON(data []byte) error {
	var aux batchResponseJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	objects := make([]ResponseObject, len(aux.Objects))
	for i, obj := range aux.Objects {
		objects[i] = obj.toResponseObject()
	}

	r.transfer = aux.Transfer
	r.objects = objects
	r.hashAlgo = aux.HashAlgo
	return nil
}

type ResponseObject struct {
	oid           string
	size          int64
	authenticated bool
	actions       *Actions
	objectError   *ObjectError
}

func NewResponseObject(
	oid string,
	size int64,
	authenticated bool,
	actions *Actions,
	objectError *ObjectError,
) ResponseObject {
	return ResponseObject{
		oid:           oid,
		size:          size,
		authenticated: authenticated,
		actions:       actions,
		objectError:   objectError,
	}
}

func (r ResponseObject) OID() string {
	return r.oid
}

func (r ResponseObject) Size() int64 {
	return r.size
}

func (r ResponseObject) Authenticated() bool {
	return r.authenticated
}

func (r ResponseObject) Actions() *Actions {
	return r.actions
}

func (r ResponseObject) Error() *ObjectError {
	return r.objectError
}

type responseObjectJSON struct {
	OID           string           `json:"oid"`
	Size          int64            `json:"size"`
	Authenticated bool             `json:"authenticated"`
	Actions       *actionsJSON     `json:"actions,omitempty"`
	Error         *objectErrorJSON `json:"error,omitempty"`
}

func (r ResponseObject) toJSON() responseObjectJSON {
	var actions *actionsJSON
	if r.actions != nil {
		aj := r.actions.toJSON()
		actions = &aj
	}

	var objectError *objectErrorJSON
	if r.objectError != nil {
		oe := r.objectError.toJSON()
		objectError = &oe
	}

	return responseObjectJSON{
		OID:           r.oid,
		Size:          r.size,
		Authenticated: r.authenticated,
		Actions:       actions,
		Error:         objectError,
	}
}

func (r responseObjectJSON) toResponseObject() ResponseObject {
	var actions *Actions
	if r.Actions != nil {
		a := r.Actions.toActions()
		actions = &a
	}

	var objectError *ObjectError
	if r.Error != nil {
		oe := r.Error.toObjectError()
		objectError = &oe
	}

	return ResponseObject{
		oid:           r.OID,
		size:          r.Size,
		authenticated: r.Authenticated,
		actions:       actions,
		objectError:   objectError,
	}
}

func (r ResponseObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.toJSON())
}

func (r *ResponseObject) UnmarshalJSON(data []byte) error {
	var aux responseObjectJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = aux.toResponseObject()
	return nil
}

type Actions struct {
	upload   *Action
	download *Action
}

func NewActions(upload *Action, download *Action) Actions {
	return Actions{
		upload:   upload,
		download: download,
	}
}

func (a Actions) Upload() *Action {
	return a.upload
}

func (a Actions) Download() *Action {
	return a.download
}

type actionsJSON struct {
	Upload   *actionJSON `json:"upload,omitempty"`
	Download *actionJSON `json:"download,omitempty"`
}

func (a Actions) toJSON() actionsJSON {
	var upload *actionJSON
	if a.upload != nil {
		uj := a.upload.toJSON()
		upload = &uj
	}

	var download *actionJSON
	if a.download != nil {
		dj := a.download.toJSON()
		download = &dj
	}

	return actionsJSON{
		Upload:   upload,
		Download: download,
	}
}

func (a actionsJSON) toActions() Actions {
	var upload *Action
	if a.Upload != nil {
		u := a.Upload.toAction()
		upload = &u
	}

	var download *Action
	if a.Download != nil {
		d := a.Download.toAction()
		download = &d
	}

	return Actions{
		upload:   upload,
		download: download,
	}
}

func (a Actions) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.toJSON())
}

func (a *Actions) UnmarshalJSON(data []byte) error {
	var aux actionsJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*a = aux.toActions()
	return nil
}

type Action struct {
	href      string
	header    map[string]string
	expiresIn int
}

func NewAction(href string, header map[string]string, expiresIn int) Action {
	headerCopy := make(map[string]string, len(header))
	for k, v := range header {
		headerCopy[k] = v
	}
	return Action{
		href:      href,
		header:    headerCopy,
		expiresIn: expiresIn,
	}
}

func (a Action) Href() string {
	return a.href
}

func (a Action) Header() map[string]string {
	result := make(map[string]string, len(a.header))
	for k, v := range a.header {
		result[k] = v
	}
	return result
}

func (a Action) ExpiresIn() int {
	return a.expiresIn
}

type actionJSON struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn int               `json:"expires_in,omitempty"`
}

func (a Action) toJSON() actionJSON {
	return actionJSON{
		Href:      a.href,
		Header:    a.header,
		ExpiresIn: a.expiresIn,
	}
}

func (a actionJSON) toAction() Action {
	return Action{
		href:      a.Href,
		header:    a.Header,
		expiresIn: a.ExpiresIn,
	}
}

func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.toJSON())
}

func (a *Action) UnmarshalJSON(data []byte) error {
	var aux actionJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*a = aux.toAction()
	return nil
}

type ObjectError struct {
	code    int
	message string
}

func NewObjectError(code int, message string) ObjectError {
	return ObjectError{
		code:    code,
		message: message,
	}
}

func (e ObjectError) Code() int {
	return e.code
}

func (e ObjectError) Message() string {
	return e.message
}

type objectErrorJSON struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e ObjectError) toJSON() objectErrorJSON {
	return objectErrorJSON{
		Code:    e.code,
		Message: e.message,
	}
}

func (e objectErrorJSON) toObjectError() ObjectError {
	return ObjectError{
		code:    e.Code,
		message: e.Message,
	}
}

func (e ObjectError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.toJSON())
}

func (e *ObjectError) UnmarshalJSON(data []byte) error {
	var aux objectErrorJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*e = aux.toObjectError()
	return nil
}
