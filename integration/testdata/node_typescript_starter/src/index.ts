console.log('Starting....');

console.log('Starting.... setting never ending timer');
let intervalCounter = 0;
function intervalFunc() { console.log(`Wahoo! I've already run ${++intervalCounter} times`);}
setInterval(intervalFunc, 1500);

console.log('Starting.... OK!');
