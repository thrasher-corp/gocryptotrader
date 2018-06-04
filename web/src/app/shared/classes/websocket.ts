export class WebSocketMessage {
    public event: string;
    public data: any;
    public exchange: string;
    public assetType: string;
    public error: string;

    public static CreateAuthenticationMessage(): WebSocketMessage {
        var response = new WebSocketMessage();
        
        response.event = WebSocketMessageType.Auth;
        response.data = { "username": window.sessionStorage["username"], "password": window.sessionStorage["password"] };

        return response;
    };

    public static GetSettingsMessage() : WebSocketMessage {
        var response = new WebSocketMessage();

        response.event = WebSocketMessageType.GetConfig;
        response.data = null;

        return response;
    }
}

export class WebSocketMessageType {
    public static Auth: string = "auth";
    public static GetConfig: string = "GetConfig";
    public static SaveConfig: string = "SaveConfig";
    public static GetPortfolio: string = "GetPortfolio";
    public static TickerUpdate: string = "ticker_update";
}
