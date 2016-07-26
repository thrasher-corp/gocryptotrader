'use strict';

angular.module('myApp.about', ['ngRoute'])

.config(['$routeProvider', function($routeProvider) {
  $routeProvider.when('/about', {
    templateUrl: '/views/about/about.html',
    controller: 'AboutController'
  });
}])

.controller('AboutController', [function() {

}]);