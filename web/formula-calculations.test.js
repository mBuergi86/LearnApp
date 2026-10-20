const test = require('node:test');
const assert = require('node:assert/strict');
const {readdirSync,readFileSync} = require('node:fs');
const {join} = require('node:path');
const R=String.raw;
const n=(x,d=3)=>String(Number(x.toFixed(d)));
const mean=(values)=>values.reduce((a,b)=>a+b,0)/values.length;
const variance=(values)=>mean(values.map(x=>(x-mean(values))**2));
const matrix=(rows)=>R`\begin{pmatrix}`+rows.map(row=>row.join('&')).join(R`\\`)+R`\end{pmatrix}`;
const input=[[1,2,0],[0,1,3],[2,1,0]],kernel=[[1,0],[0,-1]];
const convolution=[0,1].map(i=>[0,1].map(j=>kernel.reduce((total,row,a)=>total+row.reduce((sum,w,b)=>sum+w*input[i+a][j+b],0),0)));
const weights=[0.2,0.5,0.3],values=[[1,0],[0,2],[1,1]];
const context=[0,1].map(c=>weights.reduce((s,w,j)=>s+w*values[j][c],0));
const scores=[0.9,0.6,0.4,0.1],labels=[1,0,1,0];
const decisions=scores.map(p=>Number(p>=0.5));
const tp=decisions.filter((p,i)=>p===1&&labels[i]===1).length;
const fp=decisions.filter((p,i)=>p===1&&labels[i]===0).length;
const fn=decisions.filter((p,i)=>p===0&&labels[i]===1).length;
const attention=Math.exp(Math.sqrt(2))/(Math.exp(Math.sqrt(2))+1);
const x=[1,2,3],y=[2,4,6];
const covarianceSum=x.reduce((s,v,i)=>s+(v-mean(x))*(y[i]-mean(y)),0);
const correlation=covarianceSum/Math.sqrt(x.reduce((s,v)=>s+(v-mean(x))**2,0)*y.reduce((s,v)=>s+(v-mean(y))**2,0));

// These calculations do not read stored result strings or answer IDs.
// They independently reconstruct each worked example's headline result.
const expected={
  'activation-functions':R`\operatorname{ReLU}(-3)=${Math.max(0,-3)},\quad\operatorname{ReLU}(4)=${Math.max(0,4)}`,
  'artificial-neuron':R`z=${n(0.5*2+(-2)*(-1)-1)},\quad a=${n(Math.max(0,0.5*2+(-2)*(-1)-1))}`,
  'backpropagation':R`w_{\mathrm{neu}}=${n(0.5-0.1*(2*(-0.3)))},\quad b_{\mathrm{neu}}=${n(0.1-0.1*(-0.3))}`,
  'batch-iteration-epoch':R`${Math.ceil(1050/250)}\ \mathrm{Updates/Epoche}`,
  'batch-normalization':R`y=(${n(2*((1-2)/1)+0.5)},${n(2*((3-2)/1)+0.5)})`,
  'classic-attention':R`c=(${n(context[0])},${n(context[1])})`,
  'classification-threshold':R`\mathrm{Precision}=${n(tp/(tp+fp))},\quad\mathrm{Recall}=${n(tp/(tp+fn))}`,
  'cnn-convolution':R`Z=`+matrix(convolution),
  'cnn-spatial-dimensions':R`\#\mathrm{Parameter}=${6*(3*3*3+1)}`,
  'collinearity':R`(w_1,w_2)=(2,0)\ \text{oder}\ (0,1)\ \Longrightarrow\ \hat y=${2*3+0*6}`,
  'confusion-matrix':R`P\approx${n(35/(35+10))},\quad R=${n(35/(35+5))}`,
  'cross-validation':R`\mathrm{CV\!-\!Anpassungen}=${3*2*5}`,
  'data-cleaning-noise':R`\mathrm{Mittel}\approx${n(mean([2,100,4]))},\quad \mathrm{Median}=${[2,100,4].sort((a,b)=>a-b)[1]}`,
  'decision-trees-model':R`\mathrm{Gini\!-\!Gewinn}=${n(1-0.5**2-0.5**2)}`,
  'dl-loss-functions':R`\mathrm{MSE}=${mean([1**2,(-2)**2])},\quad\mathrm{MAE}=${mean([Math.abs(1),Math.abs(-2)])},\quad L_{\mathrm{CE}}\approx${n(-Math.log(0.8))}`,
  'dl-regularization':R`C=${n(0.4+0.1*(2**2+(-1)**2))}`,
  'eda-correlation-outliers':R`r=${n(correlation)}`,
  'eda-statistics':R`\mu=${mean([2,4,6])},\quad \sigma^2\approx${n(variance([2,4,6]))},\quad \sigma\approx${n(Math.sqrt(variance([2,4,6])))}`,
  'eda-workflow':R`\mathrm{Fehlend}=${15/100*100}\%`,
  'ensemble-basics':R`\hat y_{\mathrm{Reg}}=${mean([10,14,12])},\quad p_{\mathrm{soft}}=${n(mean([0.6,0.4,0.8]))}`,
  'feature-encoding':R`v=(0,1,0),\quad \mathrm{Zeitvektor}=(${Math.sin(0)},${Math.cos(0)})`,
  'feature-scaling':R`z=${(16-10)/2},\quad u=${(16-6)/(14-6)}`,
  'features-datatypes':R`N=3,\quad d=2,\quad \#\mathrm{Featurewerte}=${3*2}`,
  'gradient-descent':R`w_{\mathrm{neu}}=${n(4-0.1*8)}`,
  'hyperparameters-bias-variance':R`\mathrm{MSE}=${mean([(4-5)**2,(8-5)**2])}`,
  'k-means':R`(\mu_1,\mu_2)=(${mean([1,2])},${mean([8,9])}),\quad J=${[1,2,8,9].reduce((s,v,i)=>s+(v-(i<2?1.5:8.5))**2,0)}`,
  'linear-regression-model':R`\hat y=${2*3+1},\quad \mathrm{MSE}=${(2*3+1-8)**2}`,
  'logistic-regression-model':R`p=${n(1/(1+Math.exp(-Math.log(3))))},\quad\hat y=${Number(0.75>=0.5)}`,
  'loss-functions-ml':R`\mathrm{MSE}=${mean([(3-2)**2,(3-5)**2])},\quad L_{\mathrm{BCE}}\approx${n(-Math.log(0.8))}`,
  'lstm':R`c_{\mathrm{neu}}=${n(0.8*2+0.5*(-0.4))},\quad h_{\mathrm{neu}}\approx${n(0.9*Math.tanh(0.8*2+0.5*(-0.4)))}`,
  'missing-values':R`\mathrm{Ersatzwert}=${mean([10,20,30])}`,
  'ml-lifecycle':R`\mathrm{Fehler}=${2000*80+50000-220000}\ \mathrm{CHF}`,
  'model-training-gradient':R`w_{\mathrm{neu}}=${n(1-0.1*(-4))}`,
  'multi-head-attention':R`\mathrm{Output}=(${1+3},${2+4})`,
  'neural-network-architecture':R`\#\mathrm{Parameter}=${3*2+2}`,
  'overfitting-underfitting':R`\Delta_A=${25-1},\quad\Delta_B=${18-16}`,
  'performance-metrics':R`A=${(35+50)/100},\quad F_1\approx${n(2*35/(2*35+10+5))},\quad\mathrm{RMSE}\approx${n(Math.sqrt(mean([1,4])))}`,
  'pooling-resnet':R`y=(${2+0.5},${n(-1+0.2)})`,
  'positional-information':R`e+\mathrm{PE}(0)=(${n(0.2+Math.sin(0))},${n(0.3+Math.cos(0))},${n(-0.1+Math.sin(0))},${n(0.4+Math.cos(0))})`,
  'query-key-value':R`z\approx(${n(attention*10)},${n((1-attention)*4)})`,
  'random-forest':R`\hat y_{\mathrm{Hard}}=${[1,0,1,1,0].filter(x=>x===1).length>2?1:0},\quad\hat y_{\mathrm{Reg}}=${mean([8,10,12,9,11])}`,
  'regression-bias-term':R`\hat y=${2*4+3},\quad \hat y(0)=${2*0+3}`,
  'regression-optimization':R`w=${(1*2+2*4)/(1**2+2**2)},\quad J=${mean([(2*1-2)**2,(2*2-4)**2])}`,
  'rnn-basics':R`h_2\approx${n(Math.tanh(0.5*Math.tanh(1)))}`,
  'rnn-bptt':R`\nabla_W J=${n(0.2-0.1+0.4)}`,
  'self-attention':R`z=(${n(0.1*2+0.2*1)},${(0.7*4+0.2*1).toFixed(1)})`,
  'supervised-unsupervised':R`\mathrm{Regressionsfehler}=${11000-12000}\ \mathrm{CHF}`,
  'train-validation-test':R`(N_{\mathrm{train}},N_{\mathrm{val}},N_{\mathrm{test}})=(${1000*0.6},${1000*0.2},${1000*0.2})`,
  'vanishing-gradients':R`\sigma'(0)=${0.5*(1-0.5)},\quad0.5^{10}\approx${n(0.5**10,6)}`,
};

test('all 49 worked-example results match independently reconstructed elementary calculations',()=>{
  let checked=0;
  for(const file of readdirSync(join(__dirname,'..','content','topics'))) {
    const topic=JSON.parse(readFileSync(join(__dirname,'..','content','topics',file),'utf8'));
    if(!topic.formula) continue;
    assert.ok(expected[topic.id],topic.id+' needs an independent example check');
    assert.equal(topic.formula.worked_example.result_latex,expected[topic.id],topic.id);
    checked++;
  }
  assert.equal(checked,49);
});
