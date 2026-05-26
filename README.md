# Roboto-Go

## TODO

- Fix channel weirdness with permissions
- Fix channel id message (Unnown Message)
- Running a clear does not update the message buttons
- Improve search
    - Fix livestreams

- fix configfile for chatter

## running test instance of ollama locally

set up ollama

  curl ... 
  ollama pull <model>
  ollama serve ... 
  

## testing llama

generate

  curl -H 'Content-type: application/json' http://192.168.10.23:32300/api/generate -d @gen.test | jq

chat

  curl -H "Content-type: application/json" http://192.168.10.23:32300/api/chat -d @llama.test | jq
