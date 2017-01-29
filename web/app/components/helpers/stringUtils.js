angular.module('myApp.stringUtils', [])
  .filter('removeSpaces', [function () {
    return function (string) {
      if (!angular.isString(string)) {
        return string;
      }
      return string.replace(/[\s]/g, '');
    };
  }]);
/*
 angular.module('myApp.currenctExchangeFactory', []).factory('currentExchangeCurrency', function() {
  var currentExchangeAndCurrency = {};
  var exchangeService = {};

    exchangeService.get = function() {
      return currentExchangeAndCurrency;
    };
    exchangeService.update = function(exchange, currency) {
      currentExchangeAndCurrency.Exchange = exchange;
      currentExchangeAndCurrency.Currency = currency;
    };

    return exchangeService;
});
*/
