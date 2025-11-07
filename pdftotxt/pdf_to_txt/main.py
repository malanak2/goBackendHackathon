from flask import Flask, request, jsonify
import re
import time
import requests
from requests_toolbelt.multipart.encoder import MultipartEncoder

API_KEY = ""
BASE_URL = "https://llmwhisperer-api.us-central.unstract.com/api/v2"

app = Flask(__name__)

# ----------------------- Funkce z původního kódu -----------------------
def upload_pdf_bytes(pdf_bytes, filename="document.pdf", mode="high_quality", output_mode="text"):
    url = f"{BASE_URL}/whisper?mode={mode}&output_mode={output_mode}"
    
    m = MultipartEncoder(
        fields={
            "document": (filename, pdf_bytes, "application/pdf")
        }
    )
    
    headers = {
        "unstract-key": API_KEY,
        "Content-Type": m.content_type
    }
    
    response = requests.post(url, headers=headers, data=m)
    response.raise_for_status()
    
    return response.json()

def check_status(whisper_hash):
    url = f"{BASE_URL}/whisper-status"
    headers = {"unstract-key": API_KEY}
    params = {"whisper_hash": whisper_hash}
    
    response = requests.get(url, headers=headers, params=params)
    response.raise_for_status()
    
    return response.json()

def retrieve_text(whisper_hash):
    url = f"{BASE_URL}/whisper-retrieve"
    headers = {"unstract-key": API_KEY}
    params = {"whisper_hash": whisper_hash}
    
    response = requests.get(url, headers=headers, params=params)
    response.raise_for_status()
    
    return response.json()

def clean_text(text):
    if not text:
        return ""
    cleaned = re.sub(r'[\n\r\f\t]+', ' ', text)
    cleaned = re.sub(r'\s+', ' ', cleaned).strip()
    return cleaned

def extract_cleaned_text_from_pdf(pdf_bytes: bytes, filename="document.pdf") -> str:
    job = upload_pdf_bytes(pdf_bytes, filename)
    whisper_hash = job["whisper_hash"]

    status = ""
    while status != "processed":
        time.sleep(5)
        status_response = check_status(whisper_hash)
        status = status_response.get("status", "")

    result = retrieve_text(whisper_hash)
    raw_text = result.get("result_text", "")
    return clean_text(raw_text)

# ----------------------- Flask endpoint -----------------------
app = Flask(__name__)

@app.route('/')
def hello():
    return "Flask funguje!"

@app.route("/extract_text", methods=["POST"])
def extract_text_endpoint():
    if "file" not in request.files:
        return jsonify({"error": "Missing PDF file"}), 400
    
    file = request.files["file"]
    
    if file.filename == "":
        return jsonify({"error": "No file selected"}), 400
    
    try:
        pdf_bytes = file.read()
        cleaned_text = extract_cleaned_text_from_pdf(pdf_bytes, filename=file.filename)
        return jsonify({"text": cleaned_text})
    except Exception as e:
        return jsonify({"error": str(e)}), 500

# ----------------------- Spuštění Flask -----------------------

if __name__ == "__main__":
    app.run(debug=False, host="0.0.0.0", port=5001, use_reloader=False)
