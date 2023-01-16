Webová aplikace Rope umožňuje jednoduchým rozhraním spouštění potřebných skriptů nad naskenovanými a upravenými dokumenty. 
Aplikace slouží jako preprocessing pro následné zpracování dokumentů systémem ProArc. Aplikace běží v Dockerovém kontejneru spolu s PSQL databázi. 

Rope v tomhle momentu obsahuje dvě hlavní funkcionality: Image preparation a OCR. Dále také obsahuje správu procesů. 
Všechny dávky mají podporu pro zastavení, opětovné spuštění a změnu priority.  

Image preparation je skript, který ze vstupního .tiff souboru vygeneruje preview, thumbnail, .jpg full verzi obrázku, 
a také archivační a NDK kopii obrázku formátu JPEG2000. Tyto vygenerované soubory jsou potřebné pro rychlé načítání 
v systému ProArc a splňují všechny aktuální standardy. 

Druhou část aplikace Rope představuje skript pro generování OCR a ALTO souborů skrz PERO-OCR API. 
Na server Pera je odeslána .jpg kopie obrázku a po asynchronním zpracování na straně Pera se vrací .txt soubor 
s textovým přepisem (OCR) a .xml soubor s ALTO verzi textu. 
