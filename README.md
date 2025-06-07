Arttoy-hub
- Backend Go + Gin  
- Database MongoDB  
- Payment Omise (PromptPay)  
- Authentication: JWT in Cookies
-git clone https://github.com/watcharin28/Arttoy-hub_Back.git
- go mod download
- .env
PORT=8080
MONGO_URI=mongodb+srv://<user>:<pass>@cluster0.mongodb.net/arttoyhub_db
JWT_SECRET=your_jwt_secret
OMISE_PUBLIC_KEY=pk_test_xxx
OMISE_SECRET_KEY=sk_test_xxx
GOOGLE_APPLICATION_CREDENTIALS_JSON='{"type": "..."}'
