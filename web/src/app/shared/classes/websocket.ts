export class WebSocketMessage {
    public event: string;
    public data: object;
    public exchange: string;
    public assetType: string;

    public static CreateAuthenticationMessage(): WebSocketMessage {
        var response = new WebSocketMessage();
        
        response.event = "auth";
        response.data = { "username": window.sessionStorage["username"], "password": window.sessionStorage["password"] };

        return response;
    };
}

export class WebSocketMessageType {
    public static GetConfig: string = "GetConfig";
    public static SaveConfig: string = "SaveConfig";
    public static GetPortfolio: string = "GetPortfolio";
    public static TickerUpdate: string = "ticker_update";
}
