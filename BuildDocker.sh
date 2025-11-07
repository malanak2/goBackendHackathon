docker build -t hackathonapifin:latest .
cd pdftotext/pdf_to_text
docker build -t pdf_to_text:latest .
cd ../../txttojson/txt_to_json
docker build -t text_to_json:latest .
cd ../..
