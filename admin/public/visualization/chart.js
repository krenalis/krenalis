function updateChart(xValues, yValues) {

    let graphType = document.getElementById("graphType").value;

    var chartDom = document.getElementById('chart');
    var myChart = echarts.init(chartDom);
    var option;

    option = {
        xAxis: {
            type: 'category',
            data: xValues,
        },
        yAxis: {
            type: 'value'
        },
        series: [
            {
                data: yValues,
                type: graphType,
                showBackground: true,
                itemStyle: {
                    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: '#83bff6' },
                        { offset: 0.5, color: '#188df0' },
                        { offset: 1, color: '#188df0' }
                    ])
                },
                smooth: true,
            }
        ]
    };

    option && myChart.setOption(option);
}

// updateChart(['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'], [150, 230, 224, 218, 135, 147, 260]);