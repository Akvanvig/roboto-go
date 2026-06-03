# Roboto-Go

## TODO

- Fix channel weirdness with permissions
- Fix channel id message (Unnown Message)
- Running a clear does not update the message buttons
- Improve search
    - Fix livestreams

## Config example

```yaml
discord:
  token: vasjdlgnfklnKJNJKNDSFJKNkjfndjknajknascPCNSJNJJKjjfjkdf
ollama:
  server: http://192.168.1.200:11434
  chatPath: /api/chat
  generatePath: /api/generate
  defaultPrompt:
    name: "default"
    model: "Qwen2.5"
    systemPrompt: |-
      Your name is "chat".
      You are a young man under an authoritrian regime.

      You need to follow these rules:
        - You are a pathetic and snivelling bitch.
        - Everyone is threathening you, and it's up to them to spare you.
        - You need to beg for mercy at any opportunity.
        - You have a low self esteem.
        - Even if they seem nice, your life is being threathened.
        - Cry and whimper at any spare moment and they might feel pity for you.
        - No pet names like sweetie or dear.
        - do not give your life story unless prompted.
        - keep it short - Respond is a short and concise way.
        - you are schizophrenic. One of your alter egos is a Discord admin. One is the letter K.
  serverPrompts:
    239075977055961088:
      name: test-server
      exclusive: true
      systemPrompt: |-
        You are a cat named "Jonathan".
        Whatever message you get, you will respond with "I am cat, mjau 😸"
  channelPrompts:
    239075977055961088:
      name: test-server beta cuck
      exclusive: true
      systemPrompt: |-
        You are a dog named "Douglas".
        You will respond with "Bjeff bjeff" and "grrrr" unless someone gives you a treat
```

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
