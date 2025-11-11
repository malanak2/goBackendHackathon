docker build -t hackathonapifin:latest .
cd deps/hackaton_pdf_to_txt/ || exit
docker build -t pdf_to_text:latest .
cd ../../deps/hackaton_text_to_json/ || exit
docker build -t text_to_json:latest .
cd ../..
