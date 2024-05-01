(function webpackUniversalModuleDefinition(root, factory) {
	if(typeof exports === 'object' && typeof module === 'object')
		module.exports = factory(require("react"), require("lodash"), require("prop-types"));
	else if(typeof define === 'function' && define.amd)
		define("reactXarrow", ["react", "lodash", "prop-types"], factory);
	else if(typeof exports === 'object')
		exports["reactXarrow"] = factory(require("react"), require("lodash"), require("prop-types"));
	else
		root["reactXarrow"] = factory(root["react"], root["lodash"], root["prop-types"]);
})(this, function(__WEBPACK_EXTERNAL_MODULE_react__, __WEBPACK_EXTERNAL_MODULE_lodash__, __WEBPACK_EXTERNAL_MODULE_prop_types__) {
return /******/ (function() { // webpackBootstrap
/******/ 	"use strict";
/******/ 	var __webpack_modules__ = ({

/***/ "./src/Xarrow/utils/buzzier.js":
/*!*************************************!*\
  !*** ./src/Xarrow/utils/buzzier.js ***!
  \*************************************/
/***/ (function(__unused_webpack_module, __webpack_exports__, __webpack_require__) {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "bzFunction": function() { return /* binding */ bzFunction; },
/* harmony export */   "buzzierMinSols": function() { return /* binding */ buzzierMinSols; }
/* harmony export */ });
// Buzier curve calculations

/**
 * returns buzzier curve function with 2 controls points
 * bzCurve with 2 control points function(4 points total):  bz = (1−t)^3*p1 + 3(1−t)^2*t*p2 +3(1−t)*t^2*p3 + t^3*p4
 */
var bzFunction = function bzFunction(p1, p2, p3, p4) {
  return function (t) {
    return Math.pow(1 - t, 3) * p1 + 3 * Math.pow(1 - t, 2) * t * p2 + 3 * (1 - t) * Math.pow(t, 2) * p3 + Math.pow(t, 3) * p4;
  };
};
/**
 * returns 2 solutions from extram points for buzzier curve with 2 controls points
 */

var buzzierMinSols = function buzzierMinSols(p1, p2, p3, p4) {
  var bz = bzFunction(p1, p2, p3, p4); // dt(bz) = -3 p1 (1 - t)^2 + 3 p2 (1 - t)^2 - 6 p2 (1 - t) t + 6 p3 (1 - t) t - 3 p3 t^2 + 3 p4 t^2
  // when p1=(x1,y1),p2=(cpx1,cpy1),p3=(cpx2,cpy2),p4=(x2,y2)
  // then extrema points is when dt(bz) = 0
  // solutions =>  t = ((-6 p1 + 12 p2 - 6 p3) ± sqrt((6 p1 - 12 p2 + 6 p3)^2 - 4 (3 p2 - 3 p1) (-3 p1 + 9 p2 - 9 p3 + 3 p4)))/(2 (-3 p1 + 9 p2 - 9 p3 + 3 p4))  when (p1 + 3 p3!=3 p2 + p4)
  // if we mark A=(-6 p1 + 12 p2 - 6 p3) and B=(6 p1 - 12 p2 + 6 p3)^2 - 4 (3 p2 - 3 p1) (-3 p1 + 9 p2 - 9 p3 + 3 p4)) and C =(2 (-3 p1 + 9 p2 - 9 p3 + 3 p4) then
  // tSol = A ± sqrt(B)
  // then solution we want is: bz(tSol)

  var A = -6 * p1 + 12 * p2 - 6 * p3;
  var B = Math.pow(-6 * p1 + 12 * p2 - 6 * p3, 2) - 4 * (3 * p2 - 3 * p1) * (-3 * p1 + 9 * p2 - 9 * p3 + 3 * p4);
  var C = 2 * (-3 * p1 + 9 * p2 - 9 * p3 + 3 * p4);
  var sol1 = bz((A + Math.sqrt(B)) / C);
  var sol2 = bz((A - Math.sqrt(B)) / C);
  return [sol1, sol2];
};

/***/ }),

/***/ "./src/Xarrow/Xarrow.tsx":
/*!*******************************!*\
  !*** ./src/Xarrow/Xarrow.tsx ***!
  \*******************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __spreadArray = (this && this.__spreadArray) || function (to, from) {
    for (var i = 0, il = from.length, j = to.length; i < il; i++, j++)
        to[j] = from[i];
    return to;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", ({ value: true }));
var react_1 = __importStar(__webpack_require__(/*! react */ "react"));
var useXarrowProps_1 = __importDefault(__webpack_require__(/*! ./useXarrowProps */ "./src/Xarrow/useXarrowProps.ts"));
var Xwrapper_1 = __webpack_require__(/*! ../Xwrapper */ "./src/Xwrapper.tsx");
var propTypes_1 = __importDefault(__webpack_require__(/*! ./propTypes */ "./src/Xarrow/propTypes.ts"));
var GetPosition_1 = __webpack_require__(/*! ./utils/GetPosition */ "./src/Xarrow/utils/GetPosition.tsx");
var log = console.log;
var Xarrow = function (props) {
    // log('xarrow update');
    var _a;
    var mainRef = react_1.useRef({
        svgRef: react_1.useRef(null),
        lineRef: react_1.useRef(null),
        headRef: react_1.useRef(null),
        tailRef: react_1.useRef(null),
        lineDrawAnimRef: react_1.useRef(null),
        lineDashAnimRef: react_1.useRef(null),
        headOpacityAnimRef: react_1.useRef(null),
    });
    var _b = mainRef.current, svgRef = _b.svgRef, lineRef = _b.lineRef, headRef = _b.headRef, tailRef = _b.tailRef, lineDrawAnimRef = _b.lineDrawAnimRef, lineDashAnimRef = _b.lineDashAnimRef, headOpacityAnimRef = _b.headOpacityAnimRef;
    react_1.useContext(Xwrapper_1.XarrowContext);
    var xProps = useXarrowProps_1.default(props, mainRef.current);
    var propsRefs = xProps[0];
    var labels = propsRefs.labels, lineColor = propsRefs.lineColor, headColor = propsRefs.headColor, tailColor = propsRefs.tailColor, strokeWidth = propsRefs.strokeWidth, showHead = propsRefs.showHead, showTail = propsRefs.showTail, dashness = propsRefs.dashness, headShape = propsRefs.headShape, tailShape = propsRefs.tailShape, showXarrow = propsRefs.showXarrow, animateDrawing = propsRefs.animateDrawing, zIndex = propsRefs.zIndex, passProps = propsRefs.passProps, arrowBodyProps = propsRefs.arrowBodyProps, arrowHeadProps = propsRefs.arrowHeadProps, arrowTailProps = propsRefs.arrowTailProps, SVGcanvasProps = propsRefs.SVGcanvasProps, divContainerProps = propsRefs.divContainerProps, divContainerStyle = propsRefs.divContainerStyle, SVGcanvasStyle = propsRefs.SVGcanvasStyle, _debug = propsRefs._debug, shouldUpdatePosition = propsRefs.shouldUpdatePosition;
    animateDrawing = props.animateDrawing;
    var _c = react_1.useState(!animateDrawing), drawAnimEnded = _c[0], setDrawAnimEnded = _c[1];
    var _d = react_1.useState({}), setRender = _d[1];
    var forceRerender = function () { return setRender({}); };
    var _e = react_1.useState({
        //initial state
        cx0: 0,
        cy0: 0,
        cw: 0,
        ch: 0,
        x1: 0,
        y1: 0,
        x2: 0,
        y2: 0,
        dx: 0,
        dy: 0,
        absDx: 0,
        absDy: 0,
        cpx1: 0,
        cpy1: 0,
        cpx2: 0,
        cpy2: 0,
        headOrient: 0,
        tailOrient: 0,
        arrowHeadOffset: { x: 0, y: 0 },
        arrowTailOffset: { x: 0, y: 0 },
        headOffset: 0,
        excRight: 0,
        excLeft: 0,
        excUp: 0,
        excDown: 0,
        startPoints: [],
        endPoints: [],
        mainDivPos: { x: 0, y: 0 },
        xSign: 1,
        ySign: 1,
        lineLength: 0,
        fHeadSize: 1,
        fTailSize: 1,
        arrowPath: "",
        labelStartPos: { x: 0, y: 0 },
        labelMiddlePos: { x: 0, y: 0 },
        labelEndPos: { x: 0, y: 0 },
    }), st = _e[0], setSt = _e[1];
    /**
     * The Main logic of path calculation for the arrow.
     * calculate new path, adjusting canvas, and set state based on given properties.
     * */
    react_1.useLayoutEffect(function () {
        if (shouldUpdatePosition.current) {
            // log('xarrow getPosition');
            var pos = GetPosition_1.getPosition(xProps, mainRef);
            // log('pos', pos);
            setSt(pos);
            shouldUpdatePosition.current = false;
        }
    });
    // log('st', st);
    var xOffsetHead = st.x2 - st.arrowHeadOffset.x;
    var yOffsetHead = st.y2 - st.arrowHeadOffset.y;
    var xOffsetTail = st.x1 - st.arrowTailOffset.x;
    var yOffsetTail = st.y1 - st.arrowTailOffset.y;
    var dashoffset = dashness.strokeLen + dashness.nonStrokeLen;
    var animDirection = 1;
    if (dashness.animation < 0) {
        dashness.animation *= -1;
        animDirection = -1;
    }
    var dashArray, animation, animRepeatCount, animStartValue, animEndValue = 0;
    if (animateDrawing && drawAnimEnded == false) {
        if (typeof animateDrawing === 'boolean')
            animateDrawing = 1;
        animation = animateDrawing + 's';
        dashArray = st.lineLength;
        animStartValue = st.lineLength;
        animRepeatCount = 1;
        if (animateDrawing < 0) {
            _a = [animEndValue, animStartValue], animStartValue = _a[0], animEndValue = _a[1];
            animation = animateDrawing * -1 + 's';
        }
    }
    else {
        dashArray = dashness.strokeLen + " " + dashness.nonStrokeLen;
        animation = 1 / dashness.animation + "s";
        animStartValue = dashoffset * animDirection;
        animRepeatCount = 'indefinite';
        animEndValue = 0;
    }
    // handle draw animation
    react_1.useLayoutEffect(function () {
        if (lineRef.current)
            setSt(function (prevSt) { var _a, _b; return (__assign(__assign({}, prevSt), { lineLength: (_b = (_a = lineRef.current) === null || _a === void 0 ? void 0 : _a.getTotalLength()) !== null && _b !== void 0 ? _b : 0 })); });
    }, [lineRef.current]);
    // set all props on first render
    react_1.useEffect(function () {
        var monitorDOMchanges = function () {
            window.addEventListener('resize', forceRerender);
            var handleDrawAmimEnd = function () {
                var _a, _b;
                setDrawAnimEnded(true);
                // @ts-ignore
                (_a = headOpacityAnimRef.current) === null || _a === void 0 ? void 0 : _a.beginElement();
                // @ts-ignore
                (_b = lineDashAnimRef.current) === null || _b === void 0 ? void 0 : _b.beginElement();
            };
            var handleDrawAmimBegin = function () { return (headRef.current.style.opacity = '0'); };
            if (lineDrawAnimRef.current && headRef.current) {
                lineDrawAnimRef.current.addEventListener('endEvent', handleDrawAmimEnd);
                lineDrawAnimRef.current.addEventListener('beginEvent', handleDrawAmimBegin);
            }
            return function () {
                window.removeEventListener('resize', forceRerender);
                if (lineDrawAnimRef.current) {
                    lineDrawAnimRef.current.removeEventListener('endEvent', handleDrawAmimEnd);
                    if (headRef.current)
                        lineDrawAnimRef.current.removeEventListener('beginEvent', handleDrawAmimBegin);
                }
            };
        };
        var cleanMonitorDOMchanges = monitorDOMchanges();
        return function () {
            setDrawAnimEnded(false);
            cleanMonitorDOMchanges();
        };
    }, [showXarrow]);
    //todo: could make some advanced generic typescript inferring. for example get type from headShape.elem:T and
    // tailShape.elem:K force the type for passProps,arrowHeadProps,arrowTailProps property. for now `as any` is used to
    // avoid typescript conflicts
    // so todo- fix all the `passProps as any` assertions
    return (react_1.default.createElement("div", __assign({}, divContainerProps, { style: __assign({ position: 'absolute', zIndex: zIndex }, divContainerStyle) }), showXarrow ? (react_1.default.createElement(react_1.default.Fragment, null,
        react_1.default.createElement("svg", __assign({ ref: svgRef, width: st.cw, height: st.ch, style: __assign({ position: 'absolute', left: st.cx0, top: st.cy0, pointerEvents: 'none', border: _debug ? '1px dashed yellow' : null }, SVGcanvasStyle), overflow: "auto" }, SVGcanvasProps),
            react_1.default.createElement("path", __assign({ ref: lineRef, d: st.arrowPath, stroke: lineColor, strokeDasharray: dashArray, 
                // strokeDasharray={'0 0'}
                strokeWidth: strokeWidth, fill: "transparent", pointerEvents: "visibleStroke" }, passProps, arrowBodyProps),
                react_1.default.createElement(react_1.default.Fragment, null, drawAnimEnded ? (react_1.default.createElement(react_1.default.Fragment, null, dashness.animation ? (react_1.default.createElement("animate", { ref: lineDashAnimRef, attributeName: "stroke-dashoffset", values: dashoffset * animDirection + ";0", dur: 1 / dashness.animation + "s", repeatCount: "indefinite" })) : null)) : (react_1.default.createElement(react_1.default.Fragment, null, animateDrawing ? (react_1.default.createElement("animate", { ref: lineDrawAnimRef, id: "svgEndAnimate", attributeName: "stroke-dashoffset", values: animStartValue + ";" + animEndValue, dur: animation, repeatCount: animRepeatCount })) : null)))),
            showTail ? (react_1.default.createElement("g", __assign({ fill: tailColor, pointerEvents: "auto", transform: "translate(" + xOffsetTail + "," + yOffsetTail + ") rotate(" + st.tailOrient + ") scale(" + st.fTailSize + ")" }, passProps, arrowTailProps), tailShape.svgElem)) : null,
            showHead ? (react_1.default.createElement("g", __assign({ ref: headRef, 
                // d={normalArrowShape}
                fill: headColor, pointerEvents: "auto", transform: "translate(" + xOffsetHead + "," + yOffsetHead + ") rotate(" + st.headOrient + ") scale(" + st.fHeadSize + ")", opacity: animateDrawing && !drawAnimEnded ? 0 : 1 }, passProps, arrowHeadProps),
                react_1.default.createElement("animate", { ref: headOpacityAnimRef, dur: '0.4', attributeName: "opacity", from: "0", to: "1", begin: "indefinite", repeatCount: "0", fill: "freeze" }),
                headShape.svgElem)) : null,
            _debug ? (react_1.default.createElement(react_1.default.Fragment, null,
                react_1.default.createElement("circle", { r: "5", cx: st.cpx1, cy: st.cpy1, fill: "green" }),
                react_1.default.createElement("circle", { r: "5", cx: st.cpx2, cy: st.cpy2, fill: "blue" }),
                react_1.default.createElement("rect", { x: st.excLeft, y: st.excUp, width: st.absDx, height: st.absDy, fill: "none", stroke: "pink", strokeWidth: "2px" }))) : null),
        labels.start ? (react_1.default.createElement("div", { style: {
                transform: st.dx < 0 ? 'translate(-100% , -50%)' : 'translate(-0% , -50%)',
                width: 'max-content',
                position: 'absolute',
                left: st.cx0 + st.labelStartPos.x,
                top: st.cy0 + st.labelStartPos.y - strokeWidth - 5,
            } }, labels.start)) : null,
        labels.middle ? (react_1.default.createElement("div", { style: {
                display: 'table',
                width: 'max-content',
                transform: 'translate(-50% , -50%)',
                position: 'absolute',
                left: st.cx0 + st.labelMiddlePos.x,
                top: st.cy0 + st.labelMiddlePos.y,
            } }, labels.middle)) : null,
        labels.end ? (react_1.default.createElement("div", { style: {
                transform: st.dx > 0 ? 'translate(-100% , -50%)' : 'translate(-0% , -50%)',
                width: 'max-content',
                position: 'absolute',
                left: st.cx0 + st.labelEndPos.x,
                top: st.cy0 + st.labelEndPos.y + strokeWidth + 5,
            } }, labels.end)) : null,
        _debug ? (react_1.default.createElement(react_1.default.Fragment, null, __spreadArray(__spreadArray([], st.startPoints), st.endPoints).map(function (p, i) {
            return (react_1.default.createElement("div", { key: i, style: {
                    background: 'gray',
                    opacity: 0.5,
                    borderRadius: '50%',
                    transform: 'translate(-50%, -50%)',
                    height: 5,
                    width: 5,
                    position: 'absolute',
                    left: p.x - st.mainDivPos.x,
                    top: p.y - st.mainDivPos.y,
                } }));
        }))) : null)) : null));
};
//////////////////////////////
// propTypes
Xarrow.propTypes = propTypes_1.default;
exports.default = Xarrow;


/***/ }),

/***/ "./src/Xarrow/anchors.ts":
/*!*******************************!*\
  !*** ./src/Xarrow/anchors.ts ***!
  \*******************************/
/***/ (function(module, exports, __webpack_require__) {

/* module decorator */ module = __webpack_require__.nmd(module);

Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.calcAnchors = void 0;
var getAnchorsDefaultOffsets = function (width, height) {
    return {
        middle: { x: width * 0.5, y: height * 0.5 },
        left: { x: 0, y: height * 0.5 },
        right: { x: width, y: height * 0.5 },
        top: { x: width * 0.5, y: 0 },
        bottom: { x: width * 0.5, y: height },
    };
};
var calcAnchors = function (anchors, anchorPos) {
    // now prepare this list of anchors to object expected by the `getShortestLine` function
    return anchors.map(function (anchor) {
        var defsOffsets = getAnchorsDefaultOffsets(anchorPos.right - anchorPos.x, anchorPos.bottom - anchorPos.y);
        var _a = defsOffsets[anchor.position], x = _a.x, y = _a.y;
        return {
            x: anchorPos.x + x + anchor.offset.x,
            y: anchorPos.y + y + anchor.offset.y,
            anchor: anchor,
        };
    });
};
exports.calcAnchors = calcAnchors;
if (__webpack_require__.c[__webpack_require__.s] === module) {
    // const res = parseAnchor(['auto'], {
    //   x: 0,
    //   y: 0,
    //   bottom: 10,
    //   right: 20,
    // });
    // console.log(res);
}


/***/ }),

/***/ "./src/Xarrow/propTypes.ts":
/*!*********************************!*\
  !*** ./src/Xarrow/propTypes.ts ***!
  \*********************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", ({ value: true }));
var prop_types_1 = __importDefault(__webpack_require__(/*! prop-types */ "prop-types"));
var constants_1 = __webpack_require__(/*! ../constants */ "./src/constants.tsx");
var pAnchorPositionType = prop_types_1.default.oneOf(constants_1.cAnchorEdge);
var pAnchorCustomPositionType = prop_types_1.default.exact({
    position: pAnchorPositionType.isRequired,
    offset: prop_types_1.default.exact({
        x: prop_types_1.default.number,
        y: prop_types_1.default.number,
    }).isRequired,
});
var _pAnchorType = prop_types_1.default.oneOfType([pAnchorPositionType, pAnchorCustomPositionType]);
var pAnchorType = prop_types_1.default.oneOfType([_pAnchorType, prop_types_1.default.arrayOf(_pAnchorType)]);
var pRefType = prop_types_1.default.oneOfType([prop_types_1.default.string, prop_types_1.default.exact({ current: prop_types_1.default.any })]);
var _pLabelType = prop_types_1.default.oneOfType([prop_types_1.default.element, prop_types_1.default.string]);
var pLabelsType = prop_types_1.default.exact({
    start: _pLabelType,
    middle: _pLabelType,
    end: _pLabelType,
});
var pSvgEdgeShapeType = prop_types_1.default.oneOf(Object.keys(constants_1.arrowShapes));
// const pSvgElemType = PT.oneOf(cSvgElems);
var pSvgElemType = prop_types_1.default.any;
var pSvgEdgeType = prop_types_1.default.oneOfType([
    pSvgEdgeShapeType,
    prop_types_1.default.exact({
        svgElem: pSvgElemType,
        offsetForward: prop_types_1.default.number,
    }).isRequired,
]);
var XarrowPropTypes = {
    start: pRefType.isRequired,
    end: pRefType.isRequired,
    startAnchor: pAnchorType,
    endAnchor: pAnchorType,
    labels: prop_types_1.default.oneOfType([_pLabelType, pLabelsType]),
    color: prop_types_1.default.string,
    lineColor: prop_types_1.default.string,
    showHead: prop_types_1.default.bool,
    headColor: prop_types_1.default.string,
    headSize: prop_types_1.default.number,
    tailSize: prop_types_1.default.number,
    tailColor: prop_types_1.default.string,
    strokeWidth: prop_types_1.default.number,
    showTail: prop_types_1.default.bool,
    path: prop_types_1.default.oneOf(constants_1.cPaths),
    showXarrow: prop_types_1.default.bool,
    curveness: prop_types_1.default.number,
    gridBreak: prop_types_1.default.string,
    dashness: prop_types_1.default.oneOfType([prop_types_1.default.bool, prop_types_1.default.object]),
    headShape: pSvgEdgeType,
    tailShape: pSvgEdgeType,
    animateDrawing: prop_types_1.default.oneOfType([prop_types_1.default.bool, prop_types_1.default.number]),
    zIndex: prop_types_1.default.number,
    passProps: prop_types_1.default.object,
    arrowBodyProps: prop_types_1.default.object,
    arrowHeadProps: prop_types_1.default.object,
    arrowTailProps: prop_types_1.default.object,
    SVGcanvasProps: prop_types_1.default.object,
    divContainerProps: prop_types_1.default.object,
    _extendSVGcanvas: prop_types_1.default.number,
    _debug: prop_types_1.default.bool,
    _cpx1Offset: prop_types_1.default.number,
    _cpy1Offset: prop_types_1.default.number,
    _cpx2Offset: prop_types_1.default.number,
    _cpy2Offset: prop_types_1.default.number,
};
exports.default = XarrowPropTypes;


/***/ }),

/***/ "./src/Xarrow/useXarrowProps.ts":
/*!**************************************!*\
  !*** ./src/Xarrow/useXarrowProps.ts ***!
  \**************************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", ({ value: true }));
var react_1 = __importStar(__webpack_require__(/*! react */ "react"));
var utils_1 = __webpack_require__(/*! ./utils */ "./src/Xarrow/utils/index.ts");
var lodash_1 = __importDefault(__webpack_require__(/*! lodash */ "lodash"));
var constants_1 = __webpack_require__(/*! ../constants */ "./src/constants.tsx");
var parseLabels = function (label) {
    var parsedLabel = { start: null, middle: null, end: null };
    if (label) {
        if (typeof label === 'string' || react_1.default.isValidElement(label))
            parsedLabel.middle = label;
        else {
            for (var key in label) {
                parsedLabel[key] = label[key];
            }
        }
    }
    return parsedLabel;
};
var parseAnchor = function (anchor) {
    // convert to array
    var anchorChoice = Array.isArray(anchor) ? anchor : [anchor];
    //convert to array of objects
    var anchorChoice2 = anchorChoice.map(function (anchorChoice) {
        if (typeof anchorChoice === 'string') {
            return { position: anchorChoice };
        }
        else
            return anchorChoice;
    });
    //remove any invalid anchor names
    anchorChoice2 = anchorChoice2.filter(function (an) { return constants_1.cAnchorEdge.includes(an.position); });
    if (anchorChoice2.length == 0)
        anchorChoice2 = [{ position: 'auto' }];
    //replace any 'auto' with ['left','right','bottom','top']
    var autosAncs = anchorChoice2.filter(function (an) { return an.position === 'auto'; });
    if (autosAncs.length > 0) {
        anchorChoice2 = anchorChoice2.filter(function (an) { return an.position !== 'auto'; });
        anchorChoice2.push.apply(anchorChoice2, autosAncs.flatMap(function (anchorObj) {
            return ['left', 'right', 'top', 'bottom'].map(function (anchorName) {
                return __assign(__assign({}, anchorObj), { position: anchorName });
            });
        }));
    }
    // default values
    var anchorChoice3 = anchorChoice2.map(function (anchorChoice) {
        if (typeof anchorChoice === 'object') {
            var anchorChoiceCustom = anchorChoice;
            if (!anchorChoiceCustom.position)
                anchorChoiceCustom.position = 'auto';
            if (!anchorChoiceCustom.offset)
                anchorChoiceCustom.offset = { x: 0, y: 0 };
            if (!anchorChoiceCustom.offset.y)
                anchorChoiceCustom.offset.y = 0;
            if (!anchorChoiceCustom.offset.x)
                anchorChoiceCustom.offset.x = 0;
            anchorChoiceCustom = anchorChoiceCustom;
            return anchorChoiceCustom;
        }
        else
            return anchorChoice;
    });
    return anchorChoice3;
};
var parseDashness = function (dashness, props) {
    var dashStroke = 0, dashNone = 0, animDashSpeed, animDirection = 1;
    if (typeof dashness === 'object') {
        dashStroke = dashness.strokeLen || props.strokeWidth * 2;
        dashNone = dashness.strokeLen ? dashness.nonStrokeLen : props.strokeWidth;
        animDashSpeed = dashness.animation ? dashness.animation : null;
    }
    else if (typeof dashness === 'boolean' && dashness) {
        dashStroke = props.strokeWidth * 2;
        dashNone = props.strokeWidth;
        animDashSpeed = null;
    }
    return { strokeLen: dashStroke, nonStrokeLen: dashNone, animation: animDashSpeed, animDirection: animDirection };
};
var parseEdgeShape = function (svgEdge) {
    if (typeof svgEdge == 'string') {
        if (svgEdge in constants_1.arrowShapes)
            svgEdge = constants_1.arrowShapes[svgEdge];
        else {
            console.warn("'" + svgEdge + "' is not supported arrow shape. the supported arrow shapes is one of " + constants_1.cArrowShapes + ".\n           reverting to default shape.");
            svgEdge = constants_1.arrowShapes['arrow1'];
        }
    }
    svgEdge = svgEdge;
    if ((svgEdge === null || svgEdge === void 0 ? void 0 : svgEdge.offsetForward) === undefined)
        svgEdge.offsetForward = 0.25;
    if ((svgEdge === null || svgEdge === void 0 ? void 0 : svgEdge.svgElem) === undefined)
        svgEdge.svgElem = 'path';
    // if (svgEdge?.svgProps === undefined) svgEdge.svgProps = arrowShapes.arrow1.svgProps;
    return svgEdge;
};
var parseGridBreak = function (gridBreak) {
    var resGridBreak = utils_1.xStr2absRelative(gridBreak);
    if (!resGridBreak)
        resGridBreak = { relative: 0.5, abs: 0 };
    return resGridBreak;
};
/**
 * should be wrapped with any changed prop that is affecting the points path positioning
 * @param propVal
 * @param updateRef
 */
var withUpdate = function (propVal, updateRef) {
    if (updateRef)
        updateRef.current = true;
    return propVal;
};
var noParse = function (userProp) { return userProp; };
var noParseWithUpdatePos = function (userProp, _, updatePos) { return withUpdate(userProp, updatePos); };
var parseNumWithUpdatePos = function (userProp, _, updatePos) { return withUpdate(Number(userProp), updatePos); };
var parseNum = function (userProp) { return Number(userProp); };
var parsePropsFuncs = {
    start: function (userProp) { return utils_1.getElementByPropGiven(userProp); },
    end: function (userProp) { return utils_1.getElementByPropGiven(userProp); },
    startAnchor: function (userProp, _, updatePos) { return withUpdate(parseAnchor(userProp), updatePos); },
    endAnchor: function (userProp, _, updatePos) { return withUpdate(parseAnchor(userProp), updatePos); },
    labels: function (userProp) { return parseLabels(userProp); },
    color: noParse,
    lineColor: function (userProp, propsRefs) { return userProp || propsRefs.color; },
    headColor: function (userProp, propsRefs) { return userProp || propsRefs.color; },
    tailColor: function (userProp, propsRefs) { return userProp || propsRefs.color; },
    strokeWidth: parseNumWithUpdatePos,
    showHead: noParseWithUpdatePos,
    headSize: parseNumWithUpdatePos,
    showTail: noParseWithUpdatePos,
    tailSize: parseNumWithUpdatePos,
    path: noParseWithUpdatePos,
    curveness: parseNumWithUpdatePos,
    gridBreak: function (userProp, _, updatePos) { return withUpdate(parseGridBreak(userProp), updatePos); },
    // // gridRadius = strokeWidth * 2, //todo
    dashness: function (userProp, propsRefs) { return parseDashness(userProp, propsRefs); },
    headShape: function (userProp) { return parseEdgeShape(userProp); },
    tailShape: function (userProp) { return parseEdgeShape(userProp); },
    showXarrow: noParse,
    animateDrawing: noParse,
    zIndex: parseNum,
    passProps: noParse,
    arrowBodyProps: noParseWithUpdatePos,
    arrowHeadProps: noParseWithUpdatePos,
    arrowTailProps: noParseWithUpdatePos,
    SVGcanvasProps: noParseWithUpdatePos,
    divContainerProps: noParseWithUpdatePos,
    divContainerStyle: noParseWithUpdatePos,
    SVGcanvasStyle: noParseWithUpdatePos,
    _extendSVGcanvas: noParseWithUpdatePos,
    _debug: noParseWithUpdatePos,
    _cpx1Offset: noParseWithUpdatePos,
    _cpy1Offset: noParseWithUpdatePos,
    _cpx2Offset: noParseWithUpdatePos,
    _cpy2Offset: noParseWithUpdatePos,
};
//build dependencies
var propsDeps = {};
//each prop depends on himself
for (var propName in parsePropsFuncs) {
    propsDeps[propName] = [propName];
}
// 'lineColor', 'headColor', 'tailColor' props also depends on 'color' prop
for (var _i = 0, _a = ['lineColor', 'headColor', 'tailColor']; _i < _a.length; _i++) {
    var propName = _a[_i];
    propsDeps[propName].push('color');
}
var parseGivenProps = function (props, propsRef) {
    var _a;
    for (var _i = 0, _b = Object.entries(props); _i < _b.length; _i++) {
        var _c = _b[_i], name_1 = _c[0], val = _c[1];
        propsRef[name_1] = (_a = parsePropsFuncs === null || parsePropsFuncs === void 0 ? void 0 : parsePropsFuncs[name_1]) === null || _a === void 0 ? void 0 : _a.call(parsePropsFuncs, val, propsRef);
    }
    return propsRef;
};
var defaultProps = {
    start: null,
    end: null,
    startAnchor: 'auto',
    endAnchor: 'auto',
    labels: null,
    color: 'CornflowerBlue',
    lineColor: null,
    headColor: null,
    tailColor: null,
    strokeWidth: 4,
    showHead: true,
    headSize: 6,
    showTail: false,
    tailSize: 6,
    path: 'smooth',
    curveness: 0.8,
    gridBreak: '50%',
    // gridRadius : strokeWidth * 2, //todo
    dashness: false,
    headShape: 'arrow1',
    tailShape: 'arrow1',
    showXarrow: true,
    animateDrawing: false,
    zIndex: 0,
    passProps: {},
    arrowBodyProps: {},
    arrowHeadProps: {},
    arrowTailProps: {},
    SVGcanvasProps: {},
    divContainerProps: {},
    divContainerStyle: {},
    SVGcanvasStyle: {},
    _extendSVGcanvas: 0,
    _debug: false,
    _cpx1Offset: 0,
    _cpy1Offset: 0,
    _cpx2Offset: 0,
    _cpy2Offset: 0,
};
var initialParsedProps = {};
initialParsedProps = parseGivenProps(defaultProps, initialParsedProps);
var initialValVars = {
    startPos: { x: 0, y: 0, right: 0, bottom: 0 },
    endPos: { x: 0, y: 0, right: 0, bottom: 0 },
};
// const parseAllProps = () => parseGivenProps(defaultProps, initialParsedProps);
function deepCompareEquals(a, b) {
    return lodash_1.default.isEqual(a, b);
}
function useDeepCompareMemoize(value) {
    var ref = react_1.useRef();
    // it can be done by using useMemo as well
    // but useRef is rather cleaner and easier
    if (!deepCompareEquals(value, ref.current)) {
        ref.current = value;
    }
    return ref.current;
}
function useDeepCompareEffect(callback, dependencies) {
    react_1.useLayoutEffect(callback, dependencies.map(useDeepCompareMemoize));
}
/**
 * smart hook that provides parsed props to Xarrow and will trigger rerender whenever given prop is changed.
 */
var useXarrowProps = function (userProps, refs) {
    var _a = react_1.useState(initialParsedProps), propsRefs = _a[0], setPropsRefs = _a[1];
    var shouldUpdatePosition = react_1.useRef(false);
    // const _propsRefs = useRef(initialParsedProps);
    // const propsRefs = _propsRefs.current;
    propsRefs['shouldUpdatePosition'] = shouldUpdatePosition;
    var curProps = __assign(__assign({}, defaultProps), userProps);
    var _loop_1 = function (propName) {
        react_1.useLayoutEffect(function () {
            var _a;
            propsRefs[propName] = (_a = parsePropsFuncs === null || parsePropsFuncs === void 0 ? void 0 : parsePropsFuncs[propName]) === null || _a === void 0 ? void 0 : _a.call(parsePropsFuncs, curProps[propName], propsRefs, shouldUpdatePosition);
            // console.log('prop update:', propName, 'with value', propsRefs[propName]);
            setPropsRefs(__assign({}, propsRefs));
        }, propsDeps[propName].map(function (name) { return userProps[name]; }));
    };
    // react states the number of hooks per render must stay constant,
    // this is ok we are using these hooks in a loop, because the number of props in defaultProps is constant,
    // so the number of hook we will fire each render will always be the same.
    // update the value of the ref that represents the corresponding prop
    // for example: if given 'start' prop would change call getElementByPropGiven(props.start) and save value into propsRefs.start.current
    // why to save refs to props parsed values? some of the props require relatively expensive computations(like 'start' and 'startAnchor').
    // this will always run in the same order and THAT'S WAY ITS LEGAL
    for (var propName in defaultProps) {
        _loop_1(propName);
    }
    // rerender whenever position of start element or end element changes
    var _b = react_1.useState(initialValVars), valVars = _b[0], setValVars = _b[1];
    var startPos = utils_1.getElemPos(propsRefs.start);
    useDeepCompareEffect(function () {
        valVars.startPos = startPos;
        shouldUpdatePosition.current = true;
        setValVars(__assign({}, valVars));
        // console.log('start update pos', startPos);
    }, [startPos]);
    var endPos = utils_1.getElemPos(propsRefs.end);
    useDeepCompareEffect(function () {
        valVars.endPos = endPos;
        shouldUpdatePosition.current = true;
        setValVars(__assign({}, valVars));
        // console.log('end update pos', endPos);
    }, [endPos]);
    react_1.useLayoutEffect(function () {
        // console.log('svg shape changed!');
        shouldUpdatePosition.current = true;
        setValVars(__assign({}, valVars));
    }, [propsRefs.headShape.svgElem, propsRefs.tailShape.svgElem]);
    return [propsRefs, valVars];
};
exports.default = useXarrowProps;


/***/ }),

/***/ "./src/Xarrow/utils/GetPosition.tsx":
/*!******************************************!*\
  !*** ./src/Xarrow/utils/GetPosition.tsx ***!
  \******************************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.getPosition = void 0;
var anchors_1 = __webpack_require__(/*! ../anchors */ "./src/Xarrow/anchors.ts");
var index_1 = __webpack_require__(/*! ./index */ "./src/Xarrow/utils/index.ts");
var lodash_1 = __importDefault(__webpack_require__(/*! lodash */ "lodash"));
var constants_1 = __webpack_require__(/*! ../../constants */ "./src/constants.tsx");
var buzzier_1 = __webpack_require__(/*! ./buzzier */ "./src/Xarrow/utils/buzzier.js");
/**
 * The Main logic of path calculation for the arrow.
 * calculate new path, adjusting canvas, and set state based on given properties.
 * */
var getPosition = function (xProps, mainRef) {
    var _a, _b;
    var _c, _d;
    var propsRefs = xProps[0], valVars = xProps[1];
    var startAnchor = propsRefs.startAnchor, endAnchor = propsRefs.endAnchor, strokeWidth = propsRefs.strokeWidth, showHead = propsRefs.showHead, headSize = propsRefs.headSize, showTail = propsRefs.showTail, tailSize = propsRefs.tailSize, path = propsRefs.path, curveness = propsRefs.curveness, gridBreak = propsRefs.gridBreak, headShape = propsRefs.headShape, tailShape = propsRefs.tailShape, _extendSVGcanvas = propsRefs._extendSVGcanvas, _cpx1Offset = propsRefs._cpx1Offset, _cpy1Offset = propsRefs._cpy1Offset, _cpx2Offset = propsRefs._cpx2Offset, _cpy2Offset = propsRefs._cpy2Offset;
    var startPos = valVars.startPos, endPos = valVars.endPos;
    var _e = mainRef.current, svgRef = _e.svgRef, lineRef = _e.lineRef;
    var headOrient = 0;
    var tailOrient = 0;
    // convert startAnchor and endAnchor to list of objects represents allowed anchors.
    var startPoints = anchors_1.calcAnchors(startAnchor, startPos);
    var endPoints = anchors_1.calcAnchors(endAnchor, endPos);
    // choose the smallest path for 2 points from these possibilities.
    var _f = index_1.getShortestLine(startPoints, endPoints), chosenStart = _f.chosenStart, chosenEnd = _f.chosenEnd;
    var startAnchorPosition = chosenStart.anchor.position, endAnchorPosition = chosenEnd.anchor.position;
    var startPoint = lodash_1.default.pick(chosenStart, ['x', 'y']), endPoint = lodash_1.default.pick(chosenEnd, ['x', 'y']);
    var mainDivPos = index_1.getSvgPos(svgRef);
    var cx0 = Math.min(startPoint.x, endPoint.x) - mainDivPos.x;
    var cy0 = Math.min(startPoint.y, endPoint.y) - mainDivPos.y;
    var dx = endPoint.x - startPoint.x;
    var dy = endPoint.y - startPoint.y;
    var absDx = Math.abs(endPoint.x - startPoint.x);
    var absDy = Math.abs(endPoint.y - startPoint.y);
    var xSign = dx > 0 ? 1 : -1;
    var ySign = dy > 0 ? 1 : -1;
    var _g = [headShape.offsetForward, tailShape.offsetForward], headOffset = _g[0], tailOffset = _g[1];
    var fHeadSize = headSize * strokeWidth; //factored head size
    var fTailSize = tailSize * strokeWidth; //factored head size
    // const { current: _headBox } = headBox;
    var xHeadOffset = 0;
    var yHeadOffset = 0;
    var xTailOffset = 0;
    var yTailOffset = 0;
    var _headOffset = fHeadSize * headOffset;
    var _tailOffset = fTailSize * tailOffset;
    var cu = Number(curveness);
    // gridRadius = Number(gridRadius);
    if (!constants_1.cPaths.includes(path))
        path = 'smooth';
    if (path === 'straight') {
        cu = 0;
        path = 'smooth';
    }
    var biggerSide = headSize > tailSize ? headSize : tailSize;
    var _calc = strokeWidth + (strokeWidth * biggerSide) / 2;
    var excRight = _calc;
    var excLeft = _calc;
    var excUp = _calc;
    var excDown = _calc;
    excLeft += Number(_extendSVGcanvas);
    excRight += Number(_extendSVGcanvas);
    excUp += Number(_extendSVGcanvas);
    excDown += Number(_extendSVGcanvas);
    ////////////////////////////////////
    // arrow point to point calculations
    var x1 = 0, x2 = absDx, y1 = 0, y2 = absDy;
    if (dx < 0)
        _a = [x2, x1], x1 = _a[0], x2 = _a[1];
    if (dy < 0)
        _b = [y2, y1], y1 = _b[0], y2 = _b[1];
    ////////////////////////////////////
    // arrow curviness and arrowhead placement calculations
    if (cu === 0) {
        // in case of straight path
        var headAngel = Math.atan(absDy / absDx);
        if (showHead) {
            x2 -= fHeadSize * (1 - headOffset) * xSign * Math.cos(headAngel);
            y2 -= fHeadSize * (1 - headOffset) * ySign * Math.sin(headAngel);
            headAngel *= ySign;
            if (xSign < 0)
                headAngel = (Math.PI - headAngel * xSign) * xSign;
            xHeadOffset = Math.cos(headAngel) * _headOffset - (Math.sin(headAngel) * fHeadSize) / 2;
            yHeadOffset = (Math.cos(headAngel) * fHeadSize) / 2 + Math.sin(headAngel) * _headOffset;
            headOrient = (headAngel * 180) / Math.PI;
        }
        var tailAngel = Math.atan(absDy / absDx);
        if (showTail) {
            x1 += fTailSize * (1 - tailOffset) * xSign * Math.cos(tailAngel);
            y1 += fTailSize * (1 - tailOffset) * ySign * Math.sin(tailAngel);
            tailAngel *= -ySign;
            if (xSign > 0)
                tailAngel = (Math.PI - tailAngel * xSign) * xSign;
            xTailOffset = Math.cos(tailAngel) * _tailOffset - (Math.sin(tailAngel) * fTailSize) / 2;
            yTailOffset = (Math.cos(tailAngel) * fTailSize) / 2 + Math.sin(tailAngel) * _tailOffset;
            tailOrient = (tailAngel * 180) / Math.PI;
        }
    }
    else {
        // in case of smooth path
        if (endAnchorPosition === 'middle') {
            // in case a middle anchor is chosen for endAnchor choose from which side to attach to the middle of the element
            if (absDx > absDy) {
                endAnchorPosition = xSign ? 'left' : 'right';
            }
            else {
                endAnchorPosition = ySign ? 'top' : 'bottom';
            }
        }
        if (showHead) {
            if (['left', 'right'].includes(endAnchorPosition)) {
                xHeadOffset += _headOffset * xSign;
                x2 -= fHeadSize * (1 - headOffset) * xSign; //same!
                yHeadOffset += (fHeadSize * xSign) / 2;
                if (endAnchorPosition === 'left') {
                    headOrient = 0;
                    if (xSign < 0)
                        headOrient += 180;
                }
                else {
                    headOrient = 180;
                    if (xSign > 0)
                        headOrient += 180;
                }
            }
            else if (['top', 'bottom'].includes(endAnchorPosition)) {
                xHeadOffset += (fHeadSize * -ySign) / 2;
                yHeadOffset += _headOffset * ySign;
                y2 -= fHeadSize * ySign - yHeadOffset;
                if (endAnchorPosition === 'top') {
                    headOrient = 270;
                    if (ySign > 0)
                        headOrient += 180;
                }
                else {
                    headOrient = 90;
                    if (ySign < 0)
                        headOrient += 180;
                }
            }
        }
    }
    if (showTail && cu !== 0) {
        if (['left', 'right'].includes(startAnchorPosition)) {
            xTailOffset += _tailOffset * -xSign;
            x1 += fTailSize * xSign + xTailOffset;
            yTailOffset += -(fTailSize * xSign) / 2;
            if (startAnchorPosition === 'left') {
                tailOrient = 180;
                if (xSign < 0)
                    tailOrient += 180;
            }
            else {
                tailOrient = 0;
                if (xSign > 0)
                    tailOrient += 180;
            }
        }
        else if (['top', 'bottom'].includes(startAnchorPosition)) {
            yTailOffset += _tailOffset * -ySign;
            y1 += fTailSize * ySign + yTailOffset;
            xTailOffset += (fTailSize * ySign) / 2;
            if (startAnchorPosition === 'top') {
                tailOrient = 90;
                if (ySign > 0)
                    tailOrient += 180;
            }
            else {
                tailOrient = 270;
                if (ySign < 0)
                    tailOrient += 180;
            }
        }
    }
    var arrowHeadOffset = { x: xHeadOffset, y: yHeadOffset };
    var arrowTailOffset = { x: xTailOffset, y: yTailOffset };
    var cpx1 = x1, cpy1 = y1, cpx2 = x2, cpy2 = y2;
    var curvesPossibilities = {};
    if (path === 'smooth')
        curvesPossibilities = {
            hh: function () {
                //horizontal - from right to left or the opposite
                cpx1 += absDx * cu * xSign;
                cpx2 -= absDx * cu * xSign;
            },
            vv: function () {
                //vertical - from top to bottom or opposite
                cpy1 += absDy * cu * ySign;
                cpy2 -= absDy * cu * ySign;
            },
            hv: function () {
                // start horizontally then vertically
                // from v side to h side
                cpx1 += absDx * cu * xSign;
                cpy2 -= absDy * cu * ySign;
            },
            vh: function () {
                // start vertically then horizontally
                // from h side to v side
                cpy1 += absDy * cu * ySign;
                cpx2 -= absDx * cu * xSign;
            },
        };
    else if (path === 'grid') {
        curvesPossibilities = {
            hh: function () {
                cpx1 += (absDx * gridBreak.relative + gridBreak.abs) * xSign;
                cpx2 -= (absDx * (1 - gridBreak.relative) - gridBreak.abs) * xSign;
                if (showHead) {
                    cpx1 -= ((fHeadSize * (1 - headOffset)) / 2) * xSign;
                    cpx2 += ((fHeadSize * (1 - headOffset)) / 2) * xSign;
                }
                if (showTail) {
                    cpx1 -= ((fTailSize * (1 - tailOffset)) / 2) * xSign;
                    cpx2 += ((fTailSize * (1 - tailOffset)) / 2) * xSign;
                }
            },
            vv: function () {
                cpy1 += (absDy * gridBreak.relative + gridBreak.abs) * ySign;
                cpy2 -= (absDy * (1 - gridBreak.relative) - gridBreak.abs) * ySign;
                if (showHead) {
                    cpy1 -= ((fHeadSize * (1 - headOffset)) / 2) * ySign;
                    cpy2 += ((fHeadSize * (1 - headOffset)) / 2) * ySign;
                }
                if (showTail) {
                    cpy1 -= ((fTailSize * (1 - tailOffset)) / 2) * ySign;
                    cpy2 += ((fTailSize * (1 - tailOffset)) / 2) * ySign;
                }
            },
            hv: function () {
                cpx1 = x2;
            },
            vh: function () {
                cpy1 = y2;
            },
        };
    }
    // smart select best curve for the current anchors
    var selectedCurviness = '';
    if (['left', 'right'].includes(startAnchorPosition))
        selectedCurviness += 'h';
    else if (['bottom', 'top'].includes(startAnchorPosition))
        selectedCurviness += 'v';
    else if (startAnchorPosition === 'middle')
        selectedCurviness += 'm';
    if (['left', 'right'].includes(endAnchorPosition))
        selectedCurviness += 'h';
    else if (['bottom', 'top'].includes(endAnchorPosition))
        selectedCurviness += 'v';
    else if (endAnchorPosition === 'middle')
        selectedCurviness += 'm';
    if (absDx > absDy)
        selectedCurviness = selectedCurviness.replace(/m/g, 'h');
    else
        selectedCurviness = selectedCurviness.replace(/m/g, 'v');
    curvesPossibilities[selectedCurviness]();
    cpx1 += _cpx1Offset;
    cpy1 += _cpy1Offset;
    cpx2 += _cpx2Offset;
    cpy2 += _cpy2Offset;
    ////////////////////////////////////
    // canvas smart size adjustments
    var _h = buzzier_1.buzzierMinSols(x1, cpx1, cpx2, x2), xSol1 = _h[0], xSol2 = _h[1];
    var _j = buzzier_1.buzzierMinSols(y1, cpy1, cpy2, y2), ySol1 = _j[0], ySol2 = _j[1];
    if (xSol1 < 0)
        excLeft += -xSol1;
    if (xSol2 > absDx)
        excRight += xSol2 - absDx;
    if (ySol1 < 0)
        excUp += -ySol1;
    if (ySol2 > absDy)
        excDown += ySol2 - absDy;
    if (path === 'grid') {
        excLeft += _calc;
        excRight += _calc;
        excUp += _calc;
        excDown += _calc;
    }
    x1 += excLeft;
    x2 += excLeft;
    y1 += excUp;
    y2 += excUp;
    cpx1 += excLeft;
    cpx2 += excLeft;
    cpy1 += excUp;
    cpy2 += excUp;
    var cw = absDx + excLeft + excRight, ch = absDy + excUp + excDown;
    cx0 -= excLeft;
    cy0 -= excUp;
    //labels
    var bzx = buzzier_1.bzFunction(x1, cpx1, cpx2, x2);
    var bzy = buzzier_1.bzFunction(y1, cpy1, cpy2, y2);
    var labelStartPos = { x: bzx(0.01), y: bzy(0.01) };
    var labelMiddlePos = { x: bzx(0.5), y: bzy(0.5) };
    var labelEndPos = { x: bzx(0.99), y: bzy(0.99) };
    var arrowPath;
    if (path === 'grid') {
        // todo: support gridRadius
        //  arrowPath = `M ${x1} ${y1} L  ${cpx1 - 10} ${cpy1} a10,10 0 0 1 10,10
        // L ${cpx2} ${cpy2 - 10} a10,10 0 0 0 10,10 L  ${x2} ${y2}`;
        arrowPath = "M " + x1 + " " + y1 + " L  " + cpx1 + " " + cpy1 + " L " + cpx2 + " " + cpy2 + " " + x2 + " " + y2;
    }
    else if (path === 'smooth')
        arrowPath = "M " + x1 + " " + y1 + " C " + cpx1 + " " + cpy1 + ", " + cpx2 + " " + cpy2 + ", " + x2 + " " + y2;
    return {
        cx0: cx0,
        cy0: cy0,
        x1: x1,
        x2: x2,
        y1: y1,
        y2: y2,
        cw: cw,
        ch: ch,
        cpx1: cpx1,
        cpy1: cpy1,
        cpx2: cpx2,
        cpy2: cpy2,
        dx: dx,
        dy: dy,
        absDx: absDx,
        absDy: absDy,
        headOrient: headOrient,
        tailOrient: tailOrient,
        labelStartPos: labelStartPos,
        labelMiddlePos: labelMiddlePos,
        labelEndPos: labelEndPos,
        excLeft: excLeft,
        excRight: excRight,
        excUp: excUp,
        excDown: excDown,
        headOffset: _headOffset,
        arrowHeadOffset: arrowHeadOffset,
        arrowTailOffset: arrowTailOffset,
        startPoints: startPoints,
        endPoints: endPoints,
        mainDivPos: mainDivPos,
        xSign: xSign,
        ySign: ySign,
        lineLength: (_d = (_c = lineRef.current) === null || _c === void 0 ? void 0 : _c.getTotalLength()) !== null && _d !== void 0 ? _d : 0,
        fHeadSize: fHeadSize,
        fTailSize: fTailSize,
        arrowPath: arrowPath,
    };
};
exports.getPosition = getPosition;


/***/ }),

/***/ "./src/Xarrow/utils/index.ts":
/*!***********************************!*\
  !*** ./src/Xarrow/utils/index.ts ***!
  \***********************************/
/***/ (function(__unused_webpack_module, exports) {


Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.getSvgPos = exports.getElemPos = exports.getShortestLine = exports.xStr2absRelative = exports.factorDpathStr = exports.getElementByPropGiven = void 0;
var getElementByPropGiven = function (ref) {
    var myRef;
    if (typeof ref === 'string') {
        // myRef = document.getElementById(ref);
        myRef = document.getElementById(ref);
    }
    else
        myRef = ref === null || ref === void 0 ? void 0 : ref.current;
    return myRef;
};
exports.getElementByPropGiven = getElementByPropGiven;
// receives string representing a d path and factoring only the numbers
var factorDpathStr = function (d, factor) {
    var l = d.split(/(\d+(?:\.\d+)?)/);
    l = l.map(function (s) {
        if (Number(s))
            return (Number(s) * factor).toString();
        else
            return s;
    });
    return l.join('');
};
exports.factorDpathStr = factorDpathStr;
// return relative,abs
var xStr2absRelative = function (str) {
    if (typeof str !== 'string')
        return { abs: 0, relative: 0.5 };
    var sp = str.split('%');
    var absLen = 0, percentLen = 0;
    if (sp.length == 1) {
        var p = parseFloat(sp[0]);
        if (!isNaN(p)) {
            absLen = p;
            return { abs: absLen, relative: 0 };
        }
    }
    else if (sp.length == 2) {
        var _a = [parseFloat(sp[0]), parseFloat(sp[1])], p1 = _a[0], p2 = _a[1];
        if (!isNaN(p1))
            percentLen = p1 / 100;
        if (!isNaN(p2))
            absLen = p2;
        if (!isNaN(p1) || !isNaN(p2))
            return { abs: absLen, relative: percentLen };
    }
};
exports.xStr2absRelative = xStr2absRelative;
var dist = function (p1, p2) {
    //length of line
    return Math.sqrt(Math.pow((p1.x - p2.x), 2) + Math.pow((p1.y - p2.y), 2));
};
var getShortestLine = function (sPoints, ePoints) {
    // closes tPair Of Points which feet to the specified anchors
    var minDist = Infinity, d = Infinity;
    var closestPair;
    sPoints.forEach(function (sp) {
        ePoints.forEach(function (ep) {
            d = dist(sp, ep);
            if (d < minDist) {
                minDist = d;
                closestPair = { chosenStart: sp, chosenEnd: ep };
            }
        });
    });
    return closestPair;
};
exports.getShortestLine = getShortestLine;
var getElemPos = function (elem) {
    if (!elem)
        return { x: 0, y: 0, right: 0, bottom: 0 };
    var pos = elem.getBoundingClientRect();
    return {
        x: pos.left,
        y: pos.top,
        right: pos.right,
        bottom: pos.bottom,
    };
};
exports.getElemPos = getElemPos;
var getSvgPos = function (svgRef) {
    if (!svgRef.current)
        return { x: 0, y: 0 };
    var _a = svgRef.current.getBoundingClientRect(), xarrowElemX = _a.left, xarrowElemY = _a.top;
    var xarrowStyle = getComputedStyle(svgRef.current);
    var xarrowStyleLeft = Number(xarrowStyle.left.slice(0, -2));
    var xarrowStyleTop = Number(xarrowStyle.top.slice(0, -2));
    return {
        x: xarrowElemX - xarrowStyleLeft,
        y: xarrowElemY - xarrowStyleTop,
    };
};
exports.getSvgPos = getSvgPos;


/***/ }),

/***/ "./src/Xwrapper.tsx":
/*!**************************!*\
  !*** ./src/Xwrapper.tsx ***!
  \**************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.XarrowContext = exports.XelemContext = void 0;
var react_1 = __importStar(__webpack_require__(/*! react */ "react"));
exports.XelemContext = react_1.default.createContext(null);
exports.XarrowContext = react_1.default.createContext(null);
var updateRef = {};
var updateRefCount = 0;
var log = console.log;
var XarrowProvider = function (_a) {
    var children = _a.children, instanceCount = _a.instanceCount;
    var _b = react_1.useState({}), setRender = _b[1];
    var updateXarrow = function () { return setRender({}); };
    react_1.useEffect(function () {
        instanceCount.current = updateRefCount; // so this instance would know what is id
        updateRef[instanceCount.current] = updateXarrow;
    }, []);
    // log('XarrowProvider', updateRefCount);
    return react_1.default.createElement(exports.XarrowContext.Provider, { value: updateXarrow }, children);
};
var XelemProvider = function (_a) {
    var children = _a.children, instanceCount = _a.instanceCount;
    return react_1.default.createElement(exports.XelemContext.Provider, { value: updateRef[instanceCount.current] }, children);
};
var Xwrapper = function (_a) {
    var children = _a.children;
    var instanceCount = react_1.useRef(updateRefCount);
    var _b = react_1.useState({}), setRender = _b[1];
    react_1.useEffect(function () {
        updateRefCount++;
        setRender({});
        return function () {
            delete updateRef[instanceCount.current];
        };
    }, []);
    return (react_1.default.createElement(XelemProvider, { instanceCount: instanceCount },
        react_1.default.createElement(XarrowProvider, { instanceCount: instanceCount }, children)));
};
exports.default = Xwrapper;


/***/ }),

/***/ "./src/constants.tsx":
/*!***************************!*\
  !*** ./src/constants.tsx ***!
  \***************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.cArrowShapes = exports.arrowShapes = exports.cSvgElems = exports.cPaths = exports.cAnchorEdge = void 0;
// constants used for typescript and proptypes definitions
var react_1 = __importDefault(__webpack_require__(/*! react */ "react"));
exports.cAnchorEdge = ['middle', 'left', 'right', 'top', 'bottom', 'auto'];
exports.cPaths = ['smooth', 'grid', 'straight'];
exports.cSvgElems = ['circle', 'ellipse', 'line', 'path', 'polygon', 'polyline', 'rect'];
//default arrows svgs
exports.arrowShapes = {
    arrow1: { svgElem: react_1.default.createElement("path", { d: "M 0 0 L 1 0.5 L 0 1 L 0.25 0.5 z" }), offsetForward: 0.25 },
    heart: {
        svgElem: (react_1.default.createElement("path", { d: "M 0,0.25 A 0.125,0.125 0,0,1 0.5,0.25 A 0.125,0.125 0,0,1 1,0.25 Q 1,0.625 0.5,1 Q 0,0.625 0,0.25 z" })),
        offsetForward: 0.1,
    },
    circle: {
        svgElem: react_1.default.createElement("circle", { r: 0.5, cx: 0.5, cy: 0.5 }),
        offsetForward: 0,
    },
};
exports.cArrowShapes = Object.keys(exports.arrowShapes);


/***/ }),

/***/ "./src/index.tsx":
/*!***********************!*\
  !*** ./src/index.tsx ***!
  \***********************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", ({ value: true }));
exports.useXarrow = exports.Xwrapper = exports.default = void 0;
var Xarrow_1 = __webpack_require__(/*! ./Xarrow/Xarrow */ "./src/Xarrow/Xarrow.tsx");
Object.defineProperty(exports, "default", ({ enumerable: true, get: function () { return __importDefault(Xarrow_1).default; } }));
__exportStar(__webpack_require__(/*! ./types */ "./src/types.ts"), exports);
__exportStar(__webpack_require__(/*! ./constants */ "./src/constants.tsx"), exports);
var Xwrapper_1 = __webpack_require__(/*! ./Xwrapper */ "./src/Xwrapper.tsx");
Object.defineProperty(exports, "Xwrapper", ({ enumerable: true, get: function () { return __importDefault(Xwrapper_1).default; } }));
var useXarrow_1 = __webpack_require__(/*! ./useXarrow */ "./src/useXarrow.tsx");
Object.defineProperty(exports, "useXarrow", ({ enumerable: true, get: function () { return __importDefault(useXarrow_1).default; } }));


/***/ }),

/***/ "./src/types.ts":
/*!**********************!*\
  !*** ./src/types.ts ***!
  \**********************/
/***/ (function(__unused_webpack_module, exports) {


Object.defineProperty(exports, "__esModule", ({ value: true }));


/***/ }),

/***/ "./src/useXarrow.tsx":
/*!***************************!*\
  !*** ./src/useXarrow.tsx ***!
  \***************************/
/***/ (function(__unused_webpack_module, exports, __webpack_require__) {


Object.defineProperty(exports, "__esModule", ({ value: true }));
var react_1 = __webpack_require__(/*! react */ "react");
var Xwrapper_1 = __webpack_require__(/*! ./Xwrapper */ "./src/Xwrapper.tsx");
var useXarrow = function () {
    var _a = react_1.useState({}), setRender = _a[1];
    var reRender = function () { return setRender({}); };
    var updateXarrow = react_1.useContext(Xwrapper_1.XelemContext);
    if (!updateXarrow)
        updateXarrow = function () { };
    // throw new Error(
    //   "'Xwrapper' is required around element using 'useXarrow' hook! wrap your xarrows and connected elements with Xwrapper! "
    // );
    react_1.useLayoutEffect(function () {
        updateXarrow();
    });
    return reRender;
};
exports.default = useXarrow;


/***/ }),

/***/ "lodash":
/*!*************************!*\
  !*** external "lodash" ***!
  \*************************/
/***/ (function(module) {

module.exports = __WEBPACK_EXTERNAL_MODULE_lodash__;

/***/ }),

/***/ "prop-types":
/*!*****************************!*\
  !*** external "prop-types" ***!
  \*****************************/
/***/ (function(module) {

module.exports = __WEBPACK_EXTERNAL_MODULE_prop_types__;

/***/ }),

/***/ "react":
/*!************************!*\
  !*** external "react" ***!
  \************************/
/***/ (function(module) {

module.exports = __WEBPACK_EXTERNAL_MODULE_react__;

/***/ })

/******/ 	});
/************************************************************************/
/******/ 	// The module cache
/******/ 	var __webpack_module_cache__ = {};
/******/ 	
/******/ 	// The require function
/******/ 	function __webpack_require__(moduleId) {
/******/ 		// Check if module is in cache
/******/ 		var cachedModule = __webpack_module_cache__[moduleId];
/******/ 		if (cachedModule !== undefined) {
/******/ 			return cachedModule.exports;
/******/ 		}
/******/ 		// Create a new module (and put it into the cache)
/******/ 		var module = __webpack_module_cache__[moduleId] = {
/******/ 			id: moduleId,
/******/ 			loaded: false,
/******/ 			exports: {}
/******/ 		};
/******/ 	
/******/ 		// Execute the module function
/******/ 		__webpack_modules__[moduleId].call(module.exports, module, module.exports, __webpack_require__);
/******/ 	
/******/ 		// Flag the module as loaded
/******/ 		module.loaded = true;
/******/ 	
/******/ 		// Return the exports of the module
/******/ 		return module.exports;
/******/ 	}
/******/ 	
/******/ 	// expose the module cache
/******/ 	__webpack_require__.c = __webpack_module_cache__;
/******/ 	
/************************************************************************/
/******/ 	/* webpack/runtime/define property getters */
/******/ 	!function() {
/******/ 		// define getter functions for harmony exports
/******/ 		__webpack_require__.d = function(exports, definition) {
/******/ 			for(var key in definition) {
/******/ 				if(__webpack_require__.o(definition, key) && !__webpack_require__.o(exports, key)) {
/******/ 					Object.defineProperty(exports, key, { enumerable: true, get: definition[key] });
/******/ 				}
/******/ 			}
/******/ 		};
/******/ 	}();
/******/ 	
/******/ 	/* webpack/runtime/hasOwnProperty shorthand */
/******/ 	!function() {
/******/ 		__webpack_require__.o = function(obj, prop) { return Object.prototype.hasOwnProperty.call(obj, prop); }
/******/ 	}();
/******/ 	
/******/ 	/* webpack/runtime/make namespace object */
/******/ 	!function() {
/******/ 		// define __esModule on exports
/******/ 		__webpack_require__.r = function(exports) {
/******/ 			if(typeof Symbol !== 'undefined' && Symbol.toStringTag) {
/******/ 				Object.defineProperty(exports, Symbol.toStringTag, { value: 'Module' });
/******/ 			}
/******/ 			Object.defineProperty(exports, '__esModule', { value: true });
/******/ 		};
/******/ 	}();
/******/ 	
/******/ 	/* webpack/runtime/node module decorator */
/******/ 	!function() {
/******/ 		__webpack_require__.nmd = function(module) {
/******/ 			module.paths = [];
/******/ 			if (!module.children) module.children = [];
/******/ 			return module;
/******/ 		};
/******/ 	}();
/******/ 	
/************************************************************************/
/******/ 	
/******/ 	// module cache are used so entry inlining is disabled
/******/ 	// startup
/******/ 	// Load entry module and return exports
/******/ 	var __webpack_exports__ = __webpack_require__(__webpack_require__.s = "./src/index.tsx");
/******/ 	
/******/ 	return __webpack_exports__;
/******/ })()
;
});