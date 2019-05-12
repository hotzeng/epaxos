library(tidyr);
library(dplyr);
library(ggplot2);

data <- read.csv(file='report/plot1.csv', header=FALSE, sep=',');
colnames(data) <- c('servers', 'records', 'pipe', 'collision', 'inter', 'jitter', 'id', 'time');
data <- data %>%
	mutate(id=factor(id)) %>%
	mutate(time=10 + time / 1e6) %>%
	mutate(collision=collision / 100, jitter=jitter / 100) %>%
    group_by(id) %>%
    filter((time > quantile(time, 0.01)) & (time < quantile(time, 0.99))) %>%
	select(-servers, -records, -inter, -jitter) %>%
	mutate();

pdf('report/plot1.pdf', 5, 5);

ggplot(data, aes(x=id, y=time)) +
	geom_boxplot() + # violin(scale='area') +
	xlab('Server ID') +
	ylab('Client delay (ms)');

dev.off();
