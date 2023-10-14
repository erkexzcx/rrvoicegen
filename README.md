# rrvoicegen

RoboRock vacuum voice generator, using OpenAI ChatGPT for text-lines generation and AWS Polly service for TTS (Text To Speech).

Project is inspired (and somewhat based on) https://github.com/arner/roborock-glados ideas and instructions.

# Get started

## 0. Understand cappabilities

Visit `examples` folder in this repository and check out the audio files. It will give you a great understanding of what this project is capable off.

Also visit https://eu-central-1.console.aws.amazon.com/polly/home/ (Amazon Polly in AWS), where you can experiment with different voices.

## 1. Requirements

1. RoboRock vacuum.
2. Root (AKA "[Valetudo](https://github.com/Hypfer/Valetudo)").
3. Access to GPT3.5/GPT4 (or any other LLM as long as you can use it and get decent results).
4. AWS account. New users get free tier for 1 year after account creation, so it might be enough for you.

## 2. Generate text-lines

**NOTE**: Below are generic instructions, but what worked for me the best - using OpenAI API directly via https://bettergpt.chat/ with `GPT-4` model (everything else is default).

Create a new file `custom.csv` and open with notepad. You will store modified voice-lines there.

Using ChatGPT, generate new voice lines using `original.csv` found in this repository. Here is the ChatGPT prompt (works great with GPT4, you might want to modify it according to your preferences):

```
Below is CSV file containing robot vacuum audio definitions. You have to reply with the same CSV contents, just the 2nd column modified. You must as many different SSML tags as possible, highlighting almost every statement with different tone. Never apply SSML modifications to a whole text, only specific words or parts. Never use prosody rate or volume tags. SSML must be compatible with AWS Polly. Do not forget that voice lines are surrounded by <speak> tags in AWS Polly. Requirement for modification: Impersonate a swearing robot who is tired of dealing with owners shit. House is like a trash bin and user demands are killing you from inside. Annoy the owner!

<insert_original.csv_contents_here>
```

For example, these:

```
wifi_reset.wav,"Resetting Wi-Fi."
zone.wav,"Starting zoned cleanup."
zone_complete.wav,"Zoned cleaning completed. Going back to the dock."
```

would be turned into these:

```
wifi_reset.wav,"<speak>Resetting Wi-Fi. <prosody pitch='low'>What's next on the list?</prosody></speak>"
zone.wav,"<speak>I'm starting zoned clean up. <prosody pitch='high'>5...4...3...2...1... let's go!</prosody></speak>"
zone_complete.wav,"<speak>Finished the zoned cleaning. I'm headed back to my comfort zone, the dock.</speak>"
```

Once you have `custom.csv` file, double check if lines count is identical with `original.csv` file. At least GPT-4 never failed with such task.

## 3. Generate voice lines

Now, using AWS, generate actual voice lines in a form of audio files. This is where this application comes to help.

Download binary from here: 

Then the usage is like this:
```bash
# Export AWS environment variables (for authorization and region)
export AWS_ACCESS_KEY_ID=XXXXXXXXXXXXXXXXXXXX
export AWS_SECRET_ACCESS_KEY=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
export AWS_DEFAULT_REGION=eu-central-1

# Execute this application's help to see full list of commands
./rrvoicegen -help
Usage of ./rrvoicegen:
  -csv string
        Path to CSV file. (default "custom.csv")
  -dest string
        Dir of where generated files would be stored. (default "custom")
  -polly_engine string
        Polly engine (see https://docs.aws.amazon.com/polly/latest/dg/API_DescribeVoices.html) (default "standard")
  -polly_lang string
        Polly language (see https://docs.aws.amazon.com/polly/latest/dg/API_DescribeVoices.html) (default "en-US")
  -polly_voice string
        Polly voice (see https://docs.aws.amazon.com/polly/latest/dg/voicelist.html) (default "Matthew")

# Generate wav files using full list of command line arguments this app provides.
# If something is emitted - it will use "default" value specified in -help output.
./rrvoicegen -csv custom.csv -dest custom -polly_engine standard -polly_lang en-US -polly_voice Matthew

# Or if you wish to "override" only one command line argument, usage like this.
./rrvoicegen -polly_voice Joanna
```

Getting error like this?
```
Failed to get response from Polly: InvalidSsmlException: Unsupported SSML feature used
Failed src voice-line: <speak><say-as interpret-as='interjection'>bloody hell</say-as>, Failed to update. Restoring factory settings.  This will take about <prosody rate='slow'>5 minutes</prosody>.</speak>
exit status 1
```

Well, there is not much I can do other than suggest you to copy paste the line and go to https://eu-central-1.console.aws.amazon.com/polly/home/SynthesizeSpeech?region=eu-central-1 and try to troubleshoot which SSLM tag is _halucinated_ by an AI. Once you find it - remove it from your `custom.csv` and try generating again...

## 4. Upload to Robot vacuum

```bash
# Backup existing voices
ssh root@123.123.123.123 'cp -r /opt/rockrobo/resources/sounds/en /opt/rockrobo/resources/sounds/en_BAK'

# Override with new voices
scp -O custom/* root@123.123.123.123:/opt/rockrobo/resources/sounds/en/
```
