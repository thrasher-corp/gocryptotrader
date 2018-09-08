export class WebSocketMessageType {
    public static Auth = 'auth';
    public static GetConfig = 'GetConfig';
    public static SaveConfig = 'SaveConfig';
    public static GetPortfolio = 'GetPortfolio';
    public static TickerUpdate = 'ticker_update';
}

export class WebSocketMessage {
    public event: string;
    public data: any;
    public exchange: string;
    public assetType: string;
    public error: string;

    public static CreateAuthenticationMessage(): WebSocketMessage {
        const response = new WebSocketMessage();

        response.event = WebSocketMessageType.Auth;
        response.data = { 'username': window.sessionStorage['username'], 'password': window.sessionStorage['password'] };

        return response;
    }

    public static GetSettingsMessage(): WebSocketMessage {
        const response = new WebSocketMessage();

        response.event = WebSocketMessageType.GetConfig;
        response.data = null;

        return response;
    }
}
