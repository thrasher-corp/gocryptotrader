@echo
title GoCryptoTrader Database Model Generation
IF NOT DEFINED GOPATH (
    echo GOPATH not set
    exit
)

IF NOT DEFINED DRIVER (
    SET DRIVER=psql
)

IF %DRIVER%==psql (
    IF NOT DEFINED MODEL (SET MODEL=postgres)
) ELSE (
    IF NOT DEFINED MODEL (SET MODEL=sqlite)
)
cd ..\
start %GOPATH%\\bin\\sqlboiler -o database\\models\\%MODEL% -p %MODEL% --no-auto-timestamps %DRIVER%

pause