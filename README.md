# Crawler de Pudmed e GenBank

Esse crawler usa a biblioteca Colly para navegar pelas páginas do Pubmed e GenBank para capturar dados.

Seu uso deve ser somente para gerar um executável a ser utilizado junto com o código python do Genknowlets.

Passos:
* Na pasta root, rodar _go build_ através do terminal.
* Copiar arquivo .exe gerado para a pasta root do projeto Genknowlets.
* Na pasta root do Genknowlets, rodar _biocrawler_ através do terminal, podendo usar as flags abaixo:
    * _-u [url]_ : Define a url do Pudmed ou Genbank a ser coletada
    * _-g_ : Baixar os arquivos GBFF
    * _-r_ : Procurar outras cepas relacionadas
    * _-q_ : Esconder os logs
    * _-p_ : Printar o JSON gerado após finalização

