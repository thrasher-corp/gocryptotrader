import { Component, OnInit, OnDestroy } from '@angular/core';
import { AmChartsService, AmChart } from '@amcharts/amcharts3-angular';

@Component({
  selector: 'app-price-history',
  templateUrl: './price-history.component.html',
  styleUrls: ['./price-history.component.scss']
})
export class PriceHistoryComponent implements OnInit, OnDestroy {
  private chart: AmChart;

  public chartData = [ {
    'country': 'USA',
    'visits': 4252
  }, {
    'country': 'China',
    'visits': 1882
  }, {
    'country': 'Japan',
    'visits': 1809
  }, {
    'country': 'Germany',
    'visits': 1322
  }, {
    'country': 'UK',
    'visits': 1122
  }, {
    'country': 'France',
    'visits': 1114
  }, {
    'country': 'India',
    'visits': 984
  }, {
    'country': 'Spain',
    'visits': 711
  }, {
    'country': 'Netherlands',
    'visits': 665
  }, {
    'country': 'Russia',
    'visits': 580
  }, {
    'country': 'South Korea',
    'visits': 443
  }, {
    'country': 'Canada',
    'visits': 441
  }, {
    'country': 'Brazil',
    'visits': 395
  }, {
    'country': 'Italy',
    'visits': 386
  }, {
    'country': 'Australia',
    'visits': 384
  }, {
    'country': 'Taiwan',
    'visits': 338
  }, {
    'country': 'Poland',
    'visits': 328
} ];

  public options = {
    'type': 'serial',
  'theme': 'dark',
  'dataDateFormat': 'YYYY-MM-DD',
  'zoomOutOnDataUpdate': false,
  'valueAxes': [{
    'position': 'left'
  }],
  'graphs': [{
    'id': 'g1',
    'balloonText': 'Open:<b>[[open]]</b><br>Low:<b>[[low]]</b><br>High:<b>[[high]]</b><br>Close:<b>[[close]]</b><br>',
    'closeField': 'close',
    'fillColors': '#7f8da9',
    'highField': 'high',
    'lineColor': '#7f8da9',
    'lineAlpha': 1,
    'lowField': 'low',
    'fillAlphas': 0.9,
    'negativeFillColors': '#db4c3c',
    'negativeLineColor': '#db4c3c',
    'openField': 'open',
    'title': 'Price:',
    'type': 'candlestick',
    'valueField': 'close'
  }, {
    'valueField': 'open',
    'bullet': 'round',
    'bulletColor': '#0c0',
    'bulletAlpha': 0,
    'alphaField': 'openAlpha',
    'lineAlpha': 0,
    'showBalloon': false,
    'visibleInLegend': false
  }, {
    'valueField': 'high',
    'bullet': 'round',
    'bulletColor': '#0c0',
    'bulletAlpha': 0,
    'alphaField': 'highAlpha',
    'lineAlpha': 0,
    'showBalloon': false,
    'visibleInLegend': false
  }, {
    'valueField': 'low',
    'bullet': 'round',
    'bulletColor': '#0c0',
    'bulletAlpha': 0,
    'alphaField': 'lowAlpha',
    'lineAlpha': 0,
    'showBalloon': false,
    'visibleInLegend': false
  }, {
    'valueField': 'close',
    'bullet': 'round',
    'bulletColor': '#0c0',
    'bulletAlpha': 0,
    'alphaField': 'closeAlpha',
    'lineAlpha': 0,
    'showBalloon': false,
    'visibleInLegend': false
  }],
  'chartScrollbar': {
    'graph': 'g1',
    'graphType': 'line',
    'scrollbarHeight': 30
  },
  'chartCursor': {
    'valueLineEnabled': true,
    'valueLineBalloonEnabled': true
  },
  'categoryField': 'date',
  'categoryAxis': {
    'parseDates': true
  },
  'listeners': [{
    'event': 'clickGraphItem',
    'method': function(e) {

      // does previous bullet exist?
      if (e.chart.firstPoint !== undefined) {
        // reset
        e.item.dataContext[e.graph.alphaField] = 1;
        e.chart.firstPoint = undefined;
      } else if ( e.item.dataContext[e.graph.alphaField] === 1 ) {
        // unselect it
        e.item.dataContext[e.graph.alphaField] = undefined;
        e.chart.firstPoint = undefined;
      } else {
        e.item.dataContext[e.graph.alphaField] = 1;
        e.chart.firstPoint = e.item;
      }

      e.chart.validateData();
    }
  }],
  'dataProvider': [{
    'date': '2011-08-01',
    'open': '136.65',
    'high': '136.96',
    'low': '134.15',
    'close': '136.49'
  }, {
    'date': '2011-08-02',
    'open': '135.26',
    'high': '135.95',
    'low': '131.50',
    'close': '131.85'
  }, {
    'date': '2011-08-05',
    'open': '132.90',
    'high': '135.27',
    'low': '128.30',
    'close': '135.25'
  }, {
    'date': '2011-08-06',
    'open': '134.94',
    'high': '137.24',
    'low': '132.63',
    'close': '135.03'
  }, {
    'date': '2011-08-07',
    'open': '136.76',
    'high': '136.86',
    'low': '132.00',
    'close': '134.01'
  }, {
    'date': '2011-08-08',
    'open': '131.11',
    'high': '133.00',
    'low': '125.09',
    'close': '126.39'
  }, {
    'date': '2011-08-09',
    'open': '123.12',
    'high': '127.75',
    'low': '120.30',
    'close': '125.00'
  }, {
    'date': '2011-08-12',
    'open': '128.32',
    'high': '129.35',
    'low': '126.50',
    'close': '127.79'
  }, {
    'date': '2011-08-13',
    'open': '128.29',
    'high': '128.30',
    'low': '123.71',
    'close': '124.03'
  }, {
    'date': '2011-08-14',
    'open': '122.74',
    'high': '124.86',
    'low': '119.65',
    'close': '119.90'
  }, {
    'date': '2011-08-15',
    'open': '117.01',
    'high': '118.50',
    'low': '111.62',
    'close': '117.05'
  }, {
    'date': '2011-08-16',
    'open': '122.01',
    'high': '123.50',
    'low': '119.82',
    'close': '122.06'
  }, {
    'date': '2011-08-19',
    'open': '123.96',
    'high': '124.50',
    'low': '120.50',
    'close': '122.22'
  }, {
    'date': '2011-08-20',
    'open': '122.21',
    'high': '128.96',
    'low': '121.00',
    'close': '127.57'
  }, {
    'date': '2011-08-21',
    'open': '131.22',
    'high': '132.75',
    'low': '130.33',
    'close': '132.51'
  }, {
    'date': '2011-08-22',
    'open': '133.09',
    'high': '133.34',
    'low': '129.76',
    'close': '131.07'
  }, {
    'date': '2011-08-23',
    'open': '130.53',
    'high': '135.37',
    'low': '129.81',
    'close': '135.30'
  }, {
    'date': '2011-08-26',
    'open': '133.39',
    'high': '134.66',
    'low': '132.10',
    'close': '132.25'
  }, {
    'date': '2011-08-27',
    'open': '130.99',
    'high': '132.41',
    'low': '126.63',
    'close': '126.82'
  }, {
    'date': '2011-08-28',
    'open': '129.88',
    'high': '134.18',
    'low': '129.54',
    'close': '134.08'
  }, {
    'date': '2011-08-29',
    'open': '132.67',
    'high': '138.25',
    'low': '132.30',
    'close': '136.25'
  }, {
    'date': '2011-08-30',
    'open': '139.49',
    'high': '139.65',
    'low': '137.41',
    'close': '138.48'
  }, {
    'date': '2011-09-03',
    'open': '139.94',
    'high': '145.73',
    'low': '139.84',
    'close': '144.16'
  }, {
    'date': '2011-09-04',
    'open': '144.97',
    'high': '145.84',
    'low': '136.10',
    'close': '136.76'
  }, {
    'date': '2011-09-05',
    'open': '135.56',
    'high': '137.57',
    'low': '132.71',
    'close': '135.01'
  }, {
    'date': '2011-09-06',
    'open': '132.01',
    'high': '132.30',
    'low': '130.00',
    'close': '131.77'
  }, {
    'date': '2011-09-09',
    'open': '136.99',
    'high': '138.04',
    'low': '133.95',
    'close': '136.71'
  }, {
    'date': '2011-09-10',
    'open': '137.90',
    'high': '138.30',
    'low': '133.75',
    'close': '135.49'
  }, {
    'date': '2011-09-11',
    'open': '135.99',
    'high': '139.40',
    'low': '135.75',
    'close': '136.85'
  }, {
    'date': '2011-09-12',
    'open': '138.83',
    'high': '139.00',
    'low': '136.65',
    'close': '137.20'
  }, {
    'date': '2011-09-13',
    'open': '136.57',
    'high': '138.98',
    'low': '136.20',
    'close': '138.81'
  }, {
    'date': '2011-09-16',
    'open': '138.99',
    'high': '140.59',
    'low': '137.60',
    'close': '138.41'
  }, {
    'date': '2011-09-17',
    'open': '139.06',
    'high': '142.85',
    'low': '137.83',
    'close': '140.92'
  }, {
    'date': '2011-09-18',
    'open': '143.02',
    'high': '143.16',
    'low': '139.40',
    'close': '140.77'
  }, {
    'date': '2011-09-19',
    'open': '140.15',
    'high': '141.79',
    'low': '139.32',
    'close': '140.31'
  }, {
    'date': '2011-09-20',
    'open': '141.14',
    'high': '144.65',
    'low': '140.31',
    'close': '144.15'
  }, {
    'date': '2011-09-23',
    'open': '146.73',
    'high': '149.85',
    'low': '146.65',
    'close': '148.28'
  }, {
    'date': '2011-09-24',
    'open': '146.84',
    'high': '153.22',
    'low': '146.82',
    'close': '153.18'
  }, {
    'date': '2011-09-25',
    'open': '154.47',
    'high': '155.00',
    'low': '151.25',
    'close': '152.77'
  }, {
    'date': '2011-09-26',
    'open': '153.77',
    'high': '154.52',
    'low': '152.32',
    'close': '154.50'
  }, {
    'date': '2011-09-27',
    'open': '153.44',
    'high': '154.60',
    'low': '152.75',
    'close': '153.47'
  }, {
    'date': '2011-09-30',
    'open': '154.63',
    'high': '157.41',
    'low': '152.93',
    'close': '156.34'
  }, {
    'date': '2011-10-01',
    'open': '156.55',
    'high': '158.59',
    'low': '155.89',
    'close': '158.45'
  }, {
    'date': '2011-10-02',
    'open': '157.78',
    'high': '159.18',
    'low': '157.01',
    'close': '157.92'
  }, {
    'date': '2011-10-03',
    'open': '158.00',
    'high': '158.08',
    'low': '153.50',
    'close': '156.24'
  }, {
    'date': '2011-10-04',
    'open': '158.37',
    'high': '161.58',
    'low': '157.70',
    'close': '161.45'
  }, {
    'date': '2011-10-07',
    'open': '163.49',
    'high': '167.91',
    'low': '162.97',
    'close': '167.91'
  }, {
    'date': '2011-10-08',
    'open': '170.20',
    'high': '171.11',
    'low': '166.68',
    'close': '167.86'
  }, {
    'date': '2011-10-09',
    'open': '167.55',
    'high': '167.88',
    'low': '165.60',
    'close': '166.79'
  }]
    };




  constructor(private AmCharts: AmChartsService) { }

  ngOnInit() {
  }


  ngOnDestroy() {
    if (this.chart) {
      this.AmCharts.destroyChart(this.chart);
    }
  }

}
