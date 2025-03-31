# Whatsapp-Lite
Real-time chat application written in Go, designed for seamless and efficient communication—because sometimes, all you need is to send a message and get one back (without a hundred Slackbot reminders!)


# Getting started:

 - Set up Postgres app on macos: make postgres_setup_complete
   - While the Makefile run is sleeping, please head to Applications folder of your macos and click on the Postgres application.
   - Ensure that you click on the Initialize button before 20 seconds are up.
 - Run the main server: go run cmd/server/main.go
 - Head to http://localhost:8080 to register and login in your account
 - This application will support at least 1k concurrent users. 

# Screenshots of the current state of the application:

The screenshots of the existing state can be found at: https://github.com/codingminions/Whatsapp-Lite/tree/main/screenshots
